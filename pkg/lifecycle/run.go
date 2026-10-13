package lifecycle

import (
	"context"
	"errors"
	"fmt"

	"golang.org/x/sync/errgroup"
)

var ErrWorkerStopped = errors.New("worker stopped unexpectedly")

type Worker struct {
	Name string
	Run  func(context.Context) error
}

// Run stops all workers when one fails and waits for their cleanup before returning.
func Run(ctx context.Context, workers ...Worker) error {
	group, ctx := errgroup.WithContext(ctx)
	for _, worker := range workers {
		group.Go(func() error {
			err := worker.Run(ctx)
			if ctx.Err() != nil && (err == nil || errors.Is(err, ctx.Err())) {
				return nil
			}
			if err == nil {
				err = ErrWorkerStopped
			}
			return fmt.Errorf("%s: %w", worker.Name, err)
		})
	}
	return group.Wait()
}
