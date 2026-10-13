package lifecycle

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRunWaitsForCleanupAfterFailure(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	failure := errors.New("consumer failed")
	ready := make(chan struct{})
	cleaning := make(chan struct{})
	release := make(chan struct{})
	done := make(chan error, 1)
	go func() {
		done <- Run(ctx,
			Worker{Name: "consumer", Run: func(ctx context.Context) error { <-ready; return failure }},
			Worker{Name: "other", Run: func(ctx context.Context) error {
				close(ready)
				<-ctx.Done()
				close(cleaning)
				select {
				case <-release:
				case <-ctx.Done():
					// Cleanup has its own bounded lifetime.
					select {
					case <-release:
					case <-time.After(time.Second):
					}
				}
				return ctx.Err()
			}},
		)
	}()
	select {
	case <-cleaning:
	case <-ctx.Done():
		t.Fatal("worker was not canceled")
	}
	select {
	case <-done:
		t.Fatal("returned before cleanup")
	default:
	}
	close(release)
	select {
	case err := <-done:
		require.ErrorIs(t, err, failure)
		assert.Contains(t, err.Error(), "consumer")
	case <-ctx.Done():
		t.Fatal("workers did not stop")
	}
}

func TestRunStopsOnUnexpectedReturn(t *testing.T) {
	err := Run(context.Background(), Worker{Name: "consumer", Run: func(context.Context) error { return nil }})
	require.ErrorIs(t, err, ErrWorkerStopped)
}

func TestRunTreatsCancellationAsNormalStop(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	err := Run(ctx, Worker{Name: "consumer", Run: func(ctx context.Context) error { return ctx.Err() }})
	require.NoError(t, err)
}
