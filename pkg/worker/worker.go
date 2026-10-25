// Package worker runs a serial, cancellable background operation at a fixed delay.
package worker

import (
	"context"
	"fmt"
	"time"

	"github.com/rs/zerolog"

	"github.com/AridanWarlock/pinnAutomizer/pkg/errs"
)

type Worker interface {
	Run(ctx context.Context) error
}

type worker struct {
	interval time.Duration
	log      zerolog.Logger
	process  func(context.Context) error
}

func New(
	interval time.Duration,
	log zerolog.Logger,
	process func(context.Context) error,
) (Worker, error) {
	if interval <= 0 || process == nil {
		return nil, fmt.Errorf("%w: positive interval and worker operation required", errs.ErrInvalidArgument)
	}
	return &worker{
		interval: interval,
		log:      log,
		process:  process,
	}, nil
}

// Run executes immediately, then waits after each call, including failures.
// The owner cancels and awaits Run before closing the operation's dependencies.
func (w *worker) Run(ctx context.Context) error {
	timer := time.NewTimer(0)
	defer timer.Stop()
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-timer.C:
		}
		if err := ctx.Err(); err != nil {
			return err
		}
		err := w.process(ctx)
		if ctx.Err() != nil {
			return ctx.Err()
		}
		if err != nil {
			w.log.Error().Err(err).Msg("background operation failed; retry after interval")
		}
		timer.Reset(w.interval)
	}
}
