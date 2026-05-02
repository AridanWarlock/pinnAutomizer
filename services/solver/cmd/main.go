package main

import (
	"context"
	"fmt"
	"os/signal"
	"syscall"

	"github.com/AridanWarlock/pinnAutomizer/pkg/kafka"
	"github.com/AridanWarlock/pinnAutomizer/pkg/logger"
	"github.com/AridanWarlock/pinnAutomizer/solver/internal/adapter/mlrunner"
	"github.com/AridanWarlock/pinnAutomizer/solver/internal/config"
	tasksRun "github.com/AridanWarlock/pinnAutomizer/solver/internal/usecases/v1/tasks/run"
	"github.com/rs/zerolog"
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
	runner, err := mlrunner.NewPinnRunner(cfg.PinnRunner)
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
	onRunReader, err := kafka.NewReader(
		cfg.KafkaReader,
		"tasks.on.run",
		kafka.StrategyAtMostOnce,
		log,
	)
	if err != nil {
		return fmt.Errorf("start on run kafka reader: %w", err)
	}

	// usecases
	// solve
	runUsecase := tasksRun.New(runner)

	// consumers
	runConsumer := tasksRun.NewConsumer(runUsecase, writer)
	go func() {
		err = onRunReader.Run(ctx, runConsumer.HandleMessage)
		if err != nil {
			log.Error().Err(err).Msg("on run error")
		}
	}()

	select {
	case <-ctx.Done():
		return nil
	}
}
