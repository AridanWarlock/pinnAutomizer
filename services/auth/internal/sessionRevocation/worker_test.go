package sessionRevocation

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/rs/zerolog"
	"github.com/stretchr/testify/require"

	"github.com/AridanWarlock/pinnAutomizer/auth/internal/usecases/v1/auth/cleanupSessions"
	"github.com/AridanWarlock/pinnAutomizer/pkg/core"
)

func TestWorkerKeepsCursorOnPartialFailure(t *testing.T) {
	t.Parallel()
	next, err := core.NewJti(uuid.New())
	require.NoError(t, err)
	failure := errors.New("partial cleanup")
	uc := &MockUsecase{
		CleanupFunc: func(_ context.Context, in cleanupSessions.Input) (cleanupSessions.Output, error) {
			require.Nil(t, in.AfterJti)
			return cleanupSessions.Output{NextJti: &next}, failure
		},
	}
	w, err := NewWorker(
		uc,
		Config{
			Interval: time.Second,
			Timeout:  time.Second,
		},
		zerolog.Nop(),
	)
	require.NoError(t, err)
	require.ErrorIs(t, w.process(t.Context()), failure)
	uc.CleanupFunc = func(_ context.Context, in cleanupSessions.Input) (cleanupSessions.Output, error) {
		require.Equal(t, &next, in.AfterJti)
		return cleanupSessions.Output{}, nil
	}
	require.NoError(t, w.process(t.Context()))
	require.Nil(t, w.cursor)
}

func TestWorkerBoundsBatchContext(t *testing.T) {
	t.Parallel()
	uc := &MockUsecase{
		CleanupFunc: func(ctx context.Context, _ cleanupSessions.Input) (cleanupSessions.Output, error) {
			deadline, ok := ctx.Deadline()
			require.True(t, ok)
			require.LessOrEqual(t, time.Until(deadline), time.Second)
			return cleanupSessions.Output{}, nil
		},
	}
	w, err := NewWorker(
		uc,
		Config{
			Interval: time.Second,
			Timeout:  time.Second,
		},
		zerolog.Nop(),
	)
	require.NoError(t, err)
	require.NoError(t, w.process(t.Context()))
}

func TestWorkerDelegatesRun(t *testing.T) {
	t.Parallel()
	failure := errors.New("worker stopped")
	loop := &MockBackgroundWorker{
		RunFunc: func(ctx context.Context) error {
			require.Equal(t, t.Context(), ctx)
			return failure
		},
	}
	w := &Worker{
		worker: loop,
	}
	require.ErrorIs(t, w.Run(t.Context()), failure)
	require.Len(t, loop.RunCalls(), 1)
}
