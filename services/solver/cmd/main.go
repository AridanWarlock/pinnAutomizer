package main

import (
	"context"
	"fmt"
	"os/signal"
	"syscall"

	"github.com/rs/zerolog"

	"github.com/AridanWarlock/pinnAutomizer/pkg/kafka"
	"github.com/AridanWarlock/pinnAutomizer/pkg/lifecycle"
	"github.com/AridanWarlock/pinnAutomizer/pkg/logger"
	"github.com/AridanWarlock/pinnAutomizer/solver/internal/adapter/mlrunner"
	"github.com/AridanWarlock/pinnAutomizer/solver/internal/config"
	tasksRun "github.com/AridanWarlock/pinnAutomizer/solver/internal/usecases/v1/tasks/run"
)

// @title		PINN Automizer SolverService
// @version	1.0
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
	// pinn runner
	runner, err := mlrunner.NewPinnRunner(cfg.PinnRunner, log)
	if err != nil {
		return fmt.Errorf("start pinn runner: %w", err)
	}
	defer func() {
		if err := runner.Close(); err != nil {
			log.Error().Err(err).Msg("pinn runner shutdown error")
			return
		}
		log.Info().Msg("pinn runner shutdown gracefully")
	}()
	log.Info().Msg("pinn runner connected")
	// kafka producer
	writer := kafka.NewWriter(cfg.KafkaWriter, log)
	defer func() {
		if err := writer.Close(); err != nil {
			log.Error().Err(err).Msg("kafka producer shutdown error")
			return
		}
		log.Info().Msg("kafka producer shutdown gracefully")
	}()
	log.Info().Msg("kafka producer connected")
	// kafka reader
	runQueueReader, err := kafka.NewReader(
		cfg.KafkaReader,
		"tasks.run.queue",
		kafka.StrategyAtMostOnce,
		log,
	)
	if err != nil {
		return fmt.Errorf("start on run kafka reader: %w", err)
	}
	predictQueueReader, err := kafka.NewReader(
		cfg.KafkaReader,
		"tasks.predict.queue",
		kafka.StrategyAtMostOnce,
		log,
	)
	if err != nil {
		return fmt.Errorf("start on run kafka reader: %w", err)
	}

	// usecases
	// tasks
	runUsecase := tasksRun.New(runner)

	// consumers
	runConsumer := tasksRun.NewConsumer(runUsecase, writer)
	return lifecycle.Run(ctx,
		lifecycle.Worker{Name: "tasks.run.queue", Run: func(ctx context.Context) error {
			return runQueueReader.Run(ctx, runConsumer.HandleMessage)
		}},
		lifecycle.Worker{Name: "tasks.predict.queue", Run: func(ctx context.Context) error {
			return predictQueueReader.Run(ctx, runConsumer.HandleMessage)
		}},
	)
}
