package main

import (
	"context"
	"errors"
	"fmt"
	"os/signal"
	"syscall"

	"github.com/rs/zerolog"

	"github.com/AridanWarlock/pinnAutomizer/pkg/kafka"
	"github.com/AridanWarlock/pinnAutomizer/pkg/lifecycle"
	"github.com/AridanWarlock/pinnAutomizer/pkg/logger"
	"github.com/AridanWarlock/pinnAutomizer/solver/internal/adapter/mlrunner"
	"github.com/AridanWarlock/pinnAutomizer/solver/internal/adapter/sqlite"
	"github.com/AridanWarlock/pinnAutomizer/solver/internal/config"
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

	return run(ctx, cfg, log)
}

func run(ctx context.Context, cfg config.Config, log zerolog.Logger) (err error) {
	journal, err := sqlite.New(cfg.SQLite)
	if err != nil {
		return fmt.Errorf("open solver journal: %w", err)
	}
	defer func() { err = errors.Join(err, journal.Close()) }()
	runner, err := mlrunner.NewPinnRunner(cfg.PinnRunner, log)
	if err != nil {
		return fmt.Errorf("create pinn runner: %w", err)
	}
	defer func() { err = errors.Join(err, runner.Close()) }()
	producer := kafka.NewProducer(cfg.KafkaProducer, log)
	defer func() { err = errors.Join(err, producer.Close()) }()
	pipeline, err := newPipeline(cfg, log, journal, runner, producer)
	if err != nil {
		return err
	}
	runQueueConsumer, err := newCommandConsumer(cfg.KafkaConsumer, "tasks.run.queue", producer, log)
	if err != nil {
		return err
	}
	predictQueueConsumer, err := newCommandConsumer(cfg.KafkaConsumer, "tasks.predict.queue", producer, log)
	if err != nil {
		return err
	}
	return lifecycle.Run(ctx,
		lifecycle.Worker("tasks.run.queue", func(ctx context.Context) error {
			return runQueueConsumer.Run(ctx, pipeline.consumer.HandleMessage)
		}),
		lifecycle.Worker("tasks.predict.queue", func(ctx context.Context) error {
			return predictQueueConsumer.Run(ctx, pipeline.consumer.HandleMessage)
		}),
		lifecycle.Worker("solver.recovery", pipeline.recovery.Run),
		lifecycle.Worker("solver.outbox", pipeline.outbox.Run),
	)
}

func newCommandConsumer(cfg kafka.ConsumerConfig, topic string, producer *kafka.Producer, log zerolog.Logger) (*kafka.Consumer, error) {
	consumer, err := kafka.NewConsumer(cfg, topic, kafka.StrategyAtLeastOnce, log, kafka.WithProducer(producer))
	if err != nil {
		return nil, fmt.Errorf("create %s consumer: %w", topic, err)
	}
	return consumer, nil
}
