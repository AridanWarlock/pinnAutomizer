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
	RecoverPending(ctx context.Context, afterRunID uuid.UUID) (tasksRecover.Output, error)
}

type Worker struct {
	usecase  Usecase
	interval time.Duration
	log      zerolog.Logger
}

func NewWorker(usecase Usecase, interval time.Duration, log zerolog.Logger) (*Worker, error) {
	if usecase == nil || interval <= 0 {
		return nil, fmt.Errorf("%w: recovery usecase and positive interval required", errs.ErrInvalidArgument)
	}
	return &Worker{
		usecase:  usecase,
		interval: interval,
		log:      log,
	}, nil
}

// Run sweeps immediately, then waits between sweeps or after a listing failure.
// The owner must cancel and await Run before closing dependencies.
func (w *Worker) Run(ctx context.Context) error {
	timer := time.NewTimer(w.interval)
	defer timer.Stop()
	cursor := uuid.Nil
	for {
		if err := ctx.Err(); err != nil {
			return err
		}
		out, err := w.usecase.RecoverPending(ctx, cursor)
		if ctx.Err() != nil {
			return ctx.Err()
		}
		if err != nil {
			w.log.Error().Err(err).Msg("recover solver attempts; retry on next sweep")
		}
		advanced := out.NextRunID != cursor
		cursor = out.NextRunID
		if advanced && cursor != uuid.Nil {
			continue
		}
		timer.Reset(w.interval)
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-timer.C:
		}
	}
}
