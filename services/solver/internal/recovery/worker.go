package recovery

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/rs/zerolog"

	"github.com/AridanWarlock/pinnAutomizer/pkg/errs"
	tasksRecover "github.com/AridanWarlock/pinnAutomizer/solver/internal/usecases/v1/tasks/recover"
)

type Usecase interface {
	RecoverPending(ctx context.Context, in tasksRecover.Input) (tasksRecover.Output, error)
}

type Worker struct {
	usecase  Usecase
	interval time.Duration
	log      zerolog.Logger
}

func NewWorker(usecase Usecase, interval time.Duration, log zerolog.Logger) (*Worker, error) {
	if interval <= 0 {
		return nil, fmt.Errorf("%w: positive interval required", errs.ErrInvalidArgument)
	}
	return &Worker{
		usecase:  usecase,
		interval: interval,
		log:      log,
	}, nil
}

// Run processes its first batch immediately, then waits after every batch.
// The owner must cancel and await Run before closing dependencies.
func (w *Worker) Run(ctx context.Context) error {
	timer := time.NewTimer(w.interval)
	defer timer.Stop()
	cursor := uuid.Nil
	for {
		if err := ctx.Err(); err != nil {
			return err
		}
		in := tasksRecover.Input{AfterRunID: cursor}
		out, err := w.usecase.RecoverPending(ctx, in)
		if ctx.Err() != nil {
			return ctx.Err()
		}
		if err != nil {
			w.log.Error().Err(err).Msg("recover solver attempts; retry on next sweep")
		}
		cursor = out.NextRunID
		timer.Reset(w.interval)
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-timer.C:
		}
	}
}
