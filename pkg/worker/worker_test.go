package worker

import (
	"context"
	"errors"
	"testing"
	"testing/synctest"
	"time"

	"github.com/rs/zerolog"
	"github.com/stretchr/testify/require"

	"github.com/AridanWarlock/pinnAutomizer/pkg/errs"
)

func TestWorkerRetriesAfterDelayAndStopsOnCancellation(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		ctx, cancel := context.WithCancel(t.Context())
		defer cancel()
		calls := make(chan time.Time, 3)
		started := time.Now()
		w, err := New(time.Hour, zerolog.Nop(), func(context.Context) error {
			calls <- time.Now()
			return errors.New("temporary failure")
		})
		require.NoError(t, err)
		done := make(chan error, 1)
		go func() { done <- w.Run(ctx) }()
		require.Equal(t, started, <-calls, "first call must be immediate")
		synctest.Wait()
		require.Empty(t, calls, "failure must not cause a busy retry loop")
		require.Equal(t, started.Add(time.Hour), <-calls)
		cancel()
		require.ErrorIs(t, <-done, context.Canceled)
		require.Empty(t, calls, "no work after cancellation")
	})
}

func TestWorkerDoesNotStartWithCancelledContext(t *testing.T) {
	ctx, cancel := context.WithCancel(t.Context())
	cancel()
	calls := 0
	w, err := New(time.Second, zerolog.Nop(), func(context.Context) error {
		calls++
		return nil
	})
	require.NoError(t, err)
	require.ErrorIs(t, w.Run(ctx), context.Canceled)
	require.Zero(t, calls)
}

func TestWorkerRejectsInvalidConfiguration(t *testing.T) {
	_, err := New(0, zerolog.Nop(), func(context.Context) error { return nil })
	require.ErrorIs(t, err, errs.ErrInvalidArgument)
	_, err = New(time.Second, zerolog.Nop(), nil)
	require.ErrorIs(t, err, errs.ErrInvalidArgument)
}
