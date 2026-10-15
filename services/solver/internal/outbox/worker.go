package outbox

import (
	"context"
	"fmt"
	"time"

	"github.com/rs/zerolog"

	"github.com/AridanWarlock/pinnAutomizer/pkg/errs"
)

type Usecase interface {
	PublishPending(ctx context.Context) error
}

type Worker struct {
	usecase  Usecase
	interval time.Duration
	log      zerolog.Logger
}

func NewWorker(usecase Usecase, interval time.Duration, log zerolog.Logger) (*Worker, error) {
	if usecase == nil || interval <= 0 {
		return nil, fmt.Errorf("%w: outbox usecase and positive interval required", errs.ErrInvalidArgument)
	}
	return &Worker{usecase: usecase, interval: interval, log: log}, nil
}

// Run is owned by lifecycle: cancel, wait for Run, then close its dependencies.
func (w *Worker) Run(ctx context.Context) error {
	ticker := time.NewTicker(w.interval)
	defer ticker.Stop()
	for {
		if err := ctx.Err(); err != nil {
			return err
		}
		if err := w.usecase.PublishPending(ctx); err != nil {
			if ctx.Err() != nil {
				return ctx.Err()
			}
			w.log.Error().Err(err).Msg("publish solver outbox; retry after interval")
		}
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-ticker.C:
		}
	}
}
