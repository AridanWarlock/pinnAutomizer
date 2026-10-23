package main

import (
	"fmt"

	"github.com/rs/zerolog"

	"github.com/AridanWarlock/pinnAutomizer/pkg/core/limit"
	"github.com/AridanWarlock/pinnAutomizer/solver/internal/adapter/mlrunner"
	"github.com/AridanWarlock/pinnAutomizer/solver/internal/adapter/sqlite"
	"github.com/AridanWarlock/pinnAutomizer/solver/internal/config"
	outboxWorker "github.com/AridanWarlock/pinnAutomizer/solver/internal/outbox"
	"github.com/AridanWarlock/pinnAutomizer/solver/internal/recovery"
	solverOutbox "github.com/AridanWarlock/pinnAutomizer/solver/internal/usecases/v1/outbox"
	tasksAccept "github.com/AridanWarlock/pinnAutomizer/solver/internal/usecases/v1/tasks/accept"
	tasksCleanup "github.com/AridanWarlock/pinnAutomizer/solver/internal/usecases/v1/tasks/cleanup"
	tasksObserve "github.com/AridanWarlock/pinnAutomizer/solver/internal/usecases/v1/tasks/observe"
	tasksRecover "github.com/AridanWarlock/pinnAutomizer/solver/internal/usecases/v1/tasks/recover"
	tasksStart "github.com/AridanWarlock/pinnAutomizer/solver/internal/usecases/v1/tasks/start"
)

type pipeline struct {
	consumer *tasksAccept.Consumer
	recovery *recovery.Worker
	outbox   *outboxWorker.Worker
}

// newPipeline wires business scenarios without starting goroutines or owning dependencies.
func newPipeline(
	cfg config.Config,
	log zerolog.Logger,
	journal *sqlite.Repository,
	runner *mlrunner.PinnRunner,
	producer solverOutbox.Kafka,
) (pipeline, error) {
	batch, err := limit.New(cfg.Recovery.BatchSize)
	if err != nil {
		return pipeline{}, err
	}
	activeLimit, err := limit.New(cfg.Recovery.ActiveLimit)
	if err != nil {
		return pipeline{}, err
	}
	recoverUsecase, err := tasksRecover.New(
		journal,
		tasksStart.New(journal, runner, cfg.PinnRunner.Timeout, activeLimit),
		tasksObserve.New(journal, runner),
		tasksCleanup.New(journal, runner),
		batch,
		cfg.Recovery.AttemptTimeout,
	)
	if err != nil {
		return pipeline{}, fmt.Errorf("create recovery usecase: %w", err)
	}
	recoverWorker, err := recovery.NewWorker(recoverUsecase, cfg.Recovery.Interval, log)
	if err != nil {
		return pipeline{}, err
	}
	publishWorker, err := outboxWorker.NewWorker(solverOutbox.New(journal, producer), cfg.OutboxInterval, log)
	if err != nil {
		return pipeline{}, err
	}
	return pipeline{
		consumer: tasksAccept.NewConsumer(tasksAccept.New(journal)),
		recovery: recoverWorker,
		outbox:   publishWorker,
	}, nil
}
