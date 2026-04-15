package main

import (
	"context"
	"fmt"
	"os/signal"
	"syscall"
	"time"

	"github.com/AridanWarlock/pinnAutomizer/pinn/config"
	"github.com/AridanWarlock/pinnAutomizer/pinn/internal/adapter/postgres"
	"github.com/AridanWarlock/pinnAutomizer/pinn/internal/outbox"
	authLogin "github.com/AridanWarlock/pinnAutomizer/pinn/internal/usecases/v1/auth/login"
	authLogout "github.com/AridanWarlock/pinnAutomizer/pinn/internal/usecases/v1/auth/logout"
	authMe "github.com/AridanWarlock/pinnAutomizer/pinn/internal/usecases/v1/auth/me"
	authRefresh "github.com/AridanWarlock/pinnAutomizer/pinn/internal/usecases/v1/auth/refresh"
	authRegister "github.com/AridanWarlock/pinnAutomizer/pinn/internal/usecases/v1/auth/register"
	tasksAfterTrain "github.com/AridanWarlock/pinnAutomizer/pinn/internal/usecases/v1/task/afterTrain"
	tasksCreate "github.com/AridanWarlock/pinnAutomizer/pinn/internal/usecases/v1/task/create"
	tasksGet "github.com/AridanWarlock/pinnAutomizer/pinn/internal/usecases/v1/task/get"
	tasksOnTrain "github.com/AridanWarlock/pinnAutomizer/pinn/internal/usecases/v1/task/onTrain"
	tasksSolve "github.com/AridanWarlock/pinnAutomizer/pinn/internal/usecases/v1/task/solve"
	"github.com/AridanWarlock/pinnAutomizer/pkg/adapter/kafka"
	"github.com/AridanWarlock/pinnAutomizer/pkg/adapter/redis"
	"github.com/AridanWarlock/pinnAutomizer/pkg/adapter/redis/goRedis"
	"github.com/AridanWarlock/pinnAutomizer/pkg/adapter/redis/indempotency"
	"github.com/AridanWarlock/pinnAutomizer/pkg/jwt"
	"github.com/AridanWarlock/pinnAutomizer/pkg/logger"
	"github.com/AridanWarlock/pinnAutomizer/pkg/password"
	"github.com/AridanWarlock/pinnAutomizer/pkg/transport/http/middleware"
	"github.com/AridanWarlock/pinnAutomizer/pkg/transport/http/server"
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
	producer := kafka.NewWriter(cfg.KafkaWriter)
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
	// access token generator
	accessTokenGenerator := jwt.NewGenerator(cfg.AccessTokenGenerator)
	// password hasher
	hasher := password.NewHasher()

	// usecases
	// auth
	authLoginUsecase := authLogin.New(
		postgresAdapter,
		redisAdapter,
		accessTokenGenerator,
		hasher,
	)
	authLogoutUsecase := authLogout.New(postgresAdapter, redisAdapter)
	authMeUsecase := authMe.New(postgresAdapter)
	authRegisterUsecase := authRegister.New(postgresAdapter, hasher)
	authRefreshUsecase := authRefresh.New(postgresAdapter, redisAdapter, accessTokenGenerator)
	// tasks
	tasksCreateUsecase := tasksCreate.New(postgresAdapter, redisIdempotencyStore)
	tasksGetUsecase := tasksGet.New(postgresAdapter)
	tasksSolveUsecase := tasksSolve.New(postgresAdapter, redisIdempotencyStore)
	tasksAfterTrainUsecase := tasksAfterTrain.New(postgresAdapter, redisIdempotencyStore)
	tasksOnTrainUsecase := tasksOnTrain.New(postgresAdapter, redisIdempotencyStore)

	// http handlers
	// auth
	authLoginHandler := authLogin.NewHttpHandler(authLoginUsecase)
	authLogoutHandler := authLogout.NewHttpHandler(authLogoutUsecase)
	authMeHandler := authMe.NewHttpHandler(authMeUsecase)
	authRegisterHandler := authRegister.NewHttpHandler(authRegisterUsecase)
	authRefreshHandler := authRefresh.NewHttpHandler(authRefreshUsecase)
	// tasks
	tasksCreateHandler := tasksCreate.NewHttpHandler(tasksCreateUsecase)
	tasksGetHandler := tasksGet.NewHttpHandler(tasksGetUsecase)
	tasksSolveHandler := tasksSolve.NewHttpHandler(tasksSolveUsecase)
	// routers
	apiV1Router := server.NewApiVersionRouter(server.ApiVersion(1))
	apiV1Router.RegisterRoutes(
		// auth
		authLoginHandler.Route(),
		authLogoutHandler.Route(),
		authMeHandler.Route(),
		authRegisterHandler.Route(),
		authRefreshHandler.Route(),
		// tasks
		tasksCreateHandler.Route(),
		tasksGetHandler.Route(),
		tasksSolveHandler.Route(),
	)
	// http server
	httpServer := server.New(
		cfg.HTTP,
		log,
		middleware.RequestID(),
		middleware.Logger(log),
		middleware.TraceID(),
		middleware.Recover(),
		middleware.AuditInfo(),
	)
	httpServer.RegisterApiRouters(apiV1Router)
	// httpServer.RegisterSwagger()

	// kafka consumers
	// tasks
	tasksOnTrainConsumer := tasksOnTrain.NewConsumer(tasksOnTrainUsecase, log)
	tasksAfterTrainConsumer := tasksAfterTrain.NewConsumer(tasksAfterTrainUsecase, log)
	// tasks-on-train
	tasksOnTrainConsumeAdapter, err := kafka.NewReader(
		cfg.KafkaReader,
		"tasks.on.train",
		kafka.StrategyAtLeastOnce,
		log,
		kafka.WithWriter(producer),
	)
	if err != nil {
		return fmt.Errorf("tasks.on.train reader init: %w", err)
	}
	go func() {
		err := tasksOnTrainConsumeAdapter.Run(ctx, tasksOnTrainConsumer.HandleMessage)
		if err != nil {
			log.Error().Err(err).Msg("tasks.on.train consume error")
			return
		}
	}()
	// tasks-after-train
	tasksAfterTrainConsumeAdapter, err := kafka.NewReader(
		cfg.KafkaReader,
		"tasks.after.train",
		kafka.StrategyAtLeastOnce,
		log,
		kafka.WithWriter(producer),
	)
	if err != nil {
		return fmt.Errorf("tasks.after.train reader init: %w", err)
	}
	go func() {
		err := tasksAfterTrainConsumeAdapter.Run(ctx, tasksAfterTrainConsumer.HandleMessage)
		if err != nil {
			log.Error().Err(err).Msg("tasks.after.train consume error")
			return
		}
	}()

	return httpServer.Run(ctx)
}
