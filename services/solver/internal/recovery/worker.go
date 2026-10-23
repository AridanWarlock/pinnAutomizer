package recovery

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/rs/zerolog"

	"github.com/AridanWarlock/pinnAutomizer/pkg/errs"
	"github.com/AridanWarlock/pinnAutomizer/pkg/worker"
	tasksRecover "github.com/AridanWarlock/pinnAutomizer/solver/internal/usecases/v1/tasks/recover"
)

type Usecase interface {
	RecoverPending(ctx context.Context, in tasksRecover.Input) (tasksRecover.Output, error)
}

type Worker struct {
	usecase  Usecase
	interval time.Duration
	log      zerolog.Logger

	cursor uuid.UUID
}

func NewWorker(
	usecase Usecase,
	interval time.Duration,
	log zerolog.Logger,
) (*Worker, error) {
	if interval <= 0 {
		return nil, fmt.Errorf("%w: positive interval required", errs.ErrInvalidArgument)
	}
	return &Worker{
		usecase:  usecase,
		interval: interval,
		log:      log,
	}, nil
}

func (w *Worker) process(ctx context.Context) error {
	in := tasksRecover.Input{
		AfterRunID: w.cursor,
	}
	out, err := w.usecase.RecoverPending(ctx, in)

	w.cursor = out.NextRunID
	return err
}

// Run processes its first batch immediately, then waits after every batch.
// The owner must cancel and await Run before closing dependencies.
func (w *Worker) Run(ctx context.Context) error {
	loop, err := worker.New(w.interval, w.log, w.process)
	if err != nil {
		return err
	}

	return loop.Run(ctx)
}
