package lifecycle

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"golang.org/x/sync/errgroup"

	"github.com/AridanWarlock/pinnAutomizer/pkg/errs"
)

var ErrWorkerStopped = errors.New("worker stopped unexpectedly")

type worker struct {
	name string
	run  func(context.Context) error
}

type configuration struct {
	workers []worker
}

type Option func(*configuration) error

func Worker(name string, run func(context.Context) error) Option {
	return func(cfg *configuration) error {
		if strings.TrimSpace(name) == "" || run == nil {
			return fmt.Errorf("%w: worker %q requires a name and function", errs.ErrInvalidArgument, name)
		}
		cfg.workers = append(cfg.workers, worker{name: name, run: run})
		return nil
	}
}

// Run validates every option before starting work, then waits for cleanup.
func Run(ctx context.Context, options ...Option) error {
	var cfg configuration
	for _, option := range options {
		if option == nil {
			return fmt.Errorf("%w: nil lifecycle option", errs.ErrInvalidArgument)
		}
		if err := option(&cfg); err != nil {
			return err
		}
	}
	group, ctx := errgroup.WithContext(ctx)
	for _, worker := range cfg.workers {
		group.Go(func() error {
			err := worker.run(ctx)
			if ctx.Err() != nil && (err == nil || errors.Is(err, ctx.Err())) {
				return nil
			}
			if err == nil {
				err = ErrWorkerStopped
			}
			return fmt.Errorf("%s: %w", worker.name, err)
		})
	}
	return group.Wait()
}
