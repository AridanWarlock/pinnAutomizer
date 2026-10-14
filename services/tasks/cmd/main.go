package main

import (
	"context"
	"fmt"
	"os/signal"
	"syscall"

	"github.com/rs/zerolog"

	"github.com/AridanWarlock/pinnAutomizer/pkg/httpsrv"
	"github.com/AridanWarlock/pinnAutomizer/pkg/kafka"
	"github.com/AridanWarlock/pinnAutomizer/pkg/lifecycle"
	"github.com/AridanWarlock/pinnAutomizer/pkg/logger"
	"github.com/AridanWarlock/pinnAutomizer/tasks/internal/adapter/fileservice"
	"github.com/AridanWarlock/pinnAutomizer/tasks/internal/adapter/filestore"
	"github.com/AridanWarlock/pinnAutomizer/tasks/internal/adapter/filewriter"
	"github.com/AridanWarlock/pinnAutomizer/tasks/internal/adapter/postgres"
	"github.com/AridanWarlock/pinnAutomizer/tasks/internal/adapter/zipper"
	"github.com/AridanWarlock/pinnAutomizer/tasks/internal/config"
	"github.com/AridanWarlock/pinnAutomizer/tasks/internal/outbox"
	tasksAfterRun "github.com/AridanWarlock/pinnAutomizer/tasks/internal/usecases/v1/tasks/afterRun"
	tasksCreate "github.com/AridanWarlock/pinnAutomizer/tasks/internal/usecases/v1/tasks/create"
	tasksDelete "github.com/AridanWarlock/pinnAutomizer/tasks/internal/usecases/v1/tasks/delete"
	tasksGet "github.com/AridanWarlock/pinnAutomizer/tasks/internal/usecases/v1/tasks/get"
	tasksOnRun "github.com/AridanWarlock/pinnAutomizer/tasks/internal/usecases/v1/tasks/onRun"
	tasksPlot "github.com/AridanWarlock/pinnAutomizer/tasks/internal/usecases/v1/tasks/plot"
	tasksResults "github.com/AridanWarlock/pinnAutomizer/tasks/internal/usecases/v1/tasks/results"
	tasksRun "github.com/AridanWarlock/pinnAutomizer/tasks/internal/usecases/v1/tasks/run"
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
	postgresAdapter, err := postgres.New(cfg.Postgres, log)
	if err != nil {
		return fmt.Errorf("postgres connect: %w", err)
	}
	defer func() {
		postgresAdapter.Close()
		log.Info().Msg("postgres pool shutdown gracefully")
	}()
	log.Info().Msg("postgres connected")
	// kafka producer
	producer := kafka.NewProducer(cfg.KafkaProducer, log)
	defer func() {
		if err := producer.Close(); err != nil {
			log.Error().Err(err).Msg("kafka producer shutdown error")
			return
		}
		log.Info().Msg("kafka producer shutdown gracefully")
	}()
	log.Info().Msg("kafka_produce connected")
	// outbox worker
	outboxWorker := outbox.NewWorker(postgresAdapter, producer, log)
	defer func() {
		outboxWorker.Close()
		log.Info().Msg("outbox worker shutdown gracefully")
	}()
	log.Info().Msg("outbox worker started")
	// file store
	fileStore := filestore.NewFileStore()
	// zipper
	zip := zipper.NewZipper()
	// file writer
	fileWriter := filewriter.NewFileWriter()
	// file service
	fileService := fileservice.NewFileService()

	// usecases
	// tasks
	tasksCreateUsecase := tasksCreate.New(postgresAdapter, fileStore)
	tasksGetUsecase := tasksGet.New(postgresAdapter)
	tasksPlotUsecase := tasksPlot.New(postgresAdapter, fileWriter)
	tasksResultsUsecase := tasksResults.New(postgresAdapter, zip)
	tasksRunUsecase := tasksRun.New(postgresAdapter)
	tasksAfterRunUsecase := tasksAfterRun.New(postgresAdapter, fileService)
	tasksOnRunUsecase := tasksOnRun.New(postgresAdapter)
	tasksDeleteUsecase := tasksDelete.New(postgresAdapter, fileService)

	// http handlers
	// tasks
	tasksCreateHandler := tasksCreate.NewHttpHandler(tasksCreateUsecase)
	tasksGetHandler := tasksGet.NewHttpHandler(tasksGetUsecase)
	tasksPlotHandler := tasksPlot.NewHttpHandler(tasksPlotUsecase)
	tasksResultHandler := tasksResults.NewHttpHandler(tasksResultsUsecase)
	tasksRunHandler := tasksRun.NewHttpHandler(tasksRunUsecase)
	tasksDeleteHandler := tasksDelete.NewHttpHandler(tasksDeleteUsecase)

	// routers
	apiV1Router := httpsrv.NewApiVersionRouter(httpsrv.ApiVersion(1))
	apiV1Router.RegisterRoutes(
		// tasks
		tasksCreateHandler.Route(),
		tasksGetHandler.Route(),
		tasksPlotHandler.Route(),
		tasksResultHandler.Route(),
		tasksRunHandler.Route(),
		tasksDeleteHandler.Route(),
	)
	// http server
	server := httpsrv.NewWithDefaultMiddlewares(
		cfg.HTTP,
		log,
	)
	server.RegisterApiRouters(apiV1Router)
	// httpServer.RegisterSwagger()

	// kafka consumers
	// tasks-on-run
	tasksOnRunConsumer := tasksOnRun.NewConsumer(tasksOnRunUsecase, log)
	tasksOnRunConsumeAdapter, err := kafka.NewConsumer(
		cfg.KafkaConsumer,
		"tasks.on.run",
		kafka.StrategyAtLeastOnce,
		log,
		kafka.WithProducer(producer),
	)
	if err != nil {
		return fmt.Errorf("tasks.on.run reader init: %w", err)
	}

	// tasks-after-run
	tasksAfterRunConsumer := tasksAfterRun.NewConsumer(tasksAfterRunUsecase, log)
	tasksAfterRunConsumeAdapter, err := kafka.NewConsumer(
		cfg.KafkaConsumer,
		"tasks.after.run",
		kafka.StrategyAtLeastOnce,
		log,
		kafka.WithProducer(producer),
	)
	if err != nil {
		return fmt.Errorf("tasks.after.run reader init: %w", err)
	}

	return lifecycle.Run(ctx,
		lifecycle.Worker{Name: "http server", Run: server.Run},
		lifecycle.Worker{Name: "tasks.on.run", Run: func(ctx context.Context) error {
			return tasksOnRunConsumeAdapter.Run(ctx, tasksOnRunConsumer.HandleMessage)
		}},
		lifecycle.Worker{Name: "tasks.after.run", Run: func(ctx context.Context) error {
			return tasksAfterRunConsumeAdapter.Run(ctx, tasksAfterRunConsumer.HandleMessage)
		}},
	)
}
