package main

import (
	"context"
	"fmt"
	"os/signal"
	"syscall"
	"time"

	"github.com/AridanWarlock/pinnAutomizer/pkg/httpsrv"
	"github.com/AridanWarlock/pinnAutomizer/pkg/kafka"
	"github.com/AridanWarlock/pinnAutomizer/pkg/logger"
	"github.com/AridanWarlock/pinnAutomizer/pkg/redis"
	"github.com/AridanWarlock/pinnAutomizer/pkg/redis/goRedis"
	"github.com/AridanWarlock/pinnAutomizer/pkg/redis/indempotency"
	"github.com/AridanWarlock/pinnAutomizer/tasks/internal/adapter/filestore"
	"github.com/AridanWarlock/pinnAutomizer/tasks/internal/adapter/postgres"
	"github.com/AridanWarlock/pinnAutomizer/tasks/internal/config"
	"github.com/AridanWarlock/pinnAutomizer/tasks/internal/outbox"
	tasksAfterRun "github.com/AridanWarlock/pinnAutomizer/tasks/internal/usecases/v1/tasks/afterRun"
	tasksCreate "github.com/AridanWarlock/pinnAutomizer/tasks/internal/usecases/v1/tasks/create"
	tasksGet "github.com/AridanWarlock/pinnAutomizer/tasks/internal/usecases/v1/tasks/get"
	tasksRun "github.com/AridanWarlock/pinnAutomizer/tasks/internal/usecases/v1/tasks/run"
	"github.com/rs/zerolog"
)

// @title		PINN Automizer App
// @version	1.0
// @host		127.0.0.1:8080
// @BasePath	/api/v1
func main() {
	cfg, err := config.InitConfig()
	if err != nil {
		panic(err)
	}

	log, closeLogger, err := logger.New(cfg.Log)
	if err != nil {
		panic(err)
	}
	defer closeLogger()
	log.Info().Msg("logger configured")

	err = AppRun(cfg, log)
	if err != nil {
		panic(err)
	}
}

func AppRun(
	cfg config.Config,
	log zerolog.Logger,
) error {
	ctx, cancel := signal.NotifyContext(
		context.Background(),
		syscall.SIGINT, syscall.SIGTERM,
	)
	defer cancel()

	// adapters
	// postgres
	postgresAdapter, err := postgres.New(cfg.Postgres)
	if err != nil {
		return fmt.Errorf("postgres connect: %w", err)
	}
	defer func() {
		postgresAdapter.Close()
		log.Info().Msg("postgres pool shutdown gracefully")
	}()
	log.Info().Msg("postgres connected")
	// redis
	redisClient, err := goRedis.New(cfg.Redis)
	if err != nil {
		return fmt.Errorf("redis connect: %w", err)
	}
	defer func() {
		if err := redisClient.Close(); err != nil {
			log.Error().Err(err).Msg("shutdown redis error")
			return
		}
		log.Info().Msg("redis shutdown gracefully")
	}()
	redisAdapter := redis.NewRedis(redisClient)
	redisIdempotencyStore := indempotency.NewStore(redisAdapter, time.Hour, 3*time.Minute)
	// kafka producer
	producer := kafka.NewWriter(cfg.KafkaWriter, log)
	defer func() {
		if err := producer.Close(); err != nil {
			log.Error().Err(err).Msg("kafka producer shutdown error")
			return
		}
		log.Info().Msg("kafka producer shutdown gracefully")
	}()
	log.Info().Msg("kafka_produce connected")
	// outbox writer
	writer := outbox.NewWorker(postgresAdapter, producer, log)
	defer func() {
		writer.Close()
		log.Info().Msg("outbox writer shutdown gracefully")
	}()
	log.Info().Msg("outbox worker started")
	// file store
	fileStore := filestore.NewFileStore()

	// usecases
	// tasks
	tasksCreateUsecase := tasksCreate.New(postgresAdapter, fileStore)
	tasksGetUsecase := tasksGet.New(postgresAdapter)
	tasksRunUsecase := tasksRun.New(postgresAdapter, redisIdempotencyStore)
	tasksAfterTrainUsecase := tasksAfterRun.New(postgresAdapter, redisIdempotencyStore)

	// http handlers
	// tasks
	tasksCreateHandler := tasksCreate.NewHttpHandler(tasksCreateUsecase)
	tasksGetHandler := tasksGet.NewHttpHandler(tasksGetUsecase)
	tasksRunHandler := tasksRun.NewHttpHandler(tasksRunUsecase)
	// routers
	apiV1Router := httpsrv.NewApiVersionRouter(httpsrv.ApiVersion(1))
	apiV1Router.RegisterRoutes(
		// tasks
		tasksCreateHandler.Route(),
		tasksGetHandler.Route(),
		tasksRunHandler.Route(),
	)
	// http server
	server := httpsrv.NewWithDefaultMiddlewares(
		cfg.HTTP,
		log,
	)
	server.RegisterApiRouters(apiV1Router)
	// httpServer.RegisterSwagger()

	// kafka consumers
	// tasks-after-run
	tasksAfterTrainConsumer := tasksAfterRun.NewConsumer(tasksAfterTrainUsecase, log)
	tasksAfterTrainConsumeAdapter, err := kafka.NewReader(
		cfg.KafkaReader,
		"tasks.after.run",
		kafka.StrategyAtLeastOnce,
		log,
		kafka.WithWriter(producer),
	)
	if err != nil {
		return fmt.Errorf("tasks.after.run reader init: %w", err)
	}
	go func() {
		err := tasksAfterTrainConsumeAdapter.Run(ctx, tasksAfterTrainConsumer.HandleMessage)
		if err != nil {
			log.Error().Err(err).Msg("tasks.after.run consume error")
			return
		}
	}()

	return server.Run(ctx)
}
