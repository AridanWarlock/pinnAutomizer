package outbox

import (
	"context"
	"errors"
	"sync/atomic"
	"testing"
	"time"

	"github.com/rs/zerolog"
	"github.com/stretchr/testify/require"

	"github.com/AridanWarlock/pinnAutomizer/pkg/errs"
)

func TestWorkerRetriesAndStopsAfterCancellation(t *testing.T) {
	var calls atomic.Int32
	retried := make(chan struct{})
	usecase := &MockUsecase{PublishPendingFunc: func(ctx context.Context) error {
		if calls.Add(1) == 1 {
			return errors.New("broker unavailable")
		}
		close(retried)
		<-ctx.Done()
		return ctx.Err()
	}}
	worker, err := NewWorker(usecase, time.Millisecond, zerolog.Nop())
	require.NoError(t, err)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	done := make(chan error, 1)
	go func() { done <- worker.Run(ctx) }()
	select {
	case <-retried:
	case <-ctx.Done():
		t.Fatal("worker did not retry")
	}
	cancel()
	require.ErrorIs(t, <-done, context.Canceled)
	require.EqualValues(t, 2, calls.Load())
}

func TestWorkerRejectsInvalidDependenciesAndInterval(t *testing.T) {
	_, err := NewWorker(nil, time.Second, zerolog.Nop())
	require.ErrorIs(t, err, errs.ErrInvalidArgument)
	_, err = NewWorker(&MockUsecase{}, 0, zerolog.Nop())
	require.ErrorIs(t, err, errs.ErrInvalidArgument)
}
