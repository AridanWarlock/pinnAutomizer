package recovery

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/rs/zerolog"
	"github.com/stretchr/testify/require"

	"github.com/AridanWarlock/pinnAutomizer/pkg/errs"
	tasksRecover "github.com/AridanWarlock/pinnAutomizer/solver/internal/usecases/v1/tasks/recover"
)

func TestWorkerSweepsImmediatelyDespiteAttemptFailure(t *testing.T) {
	cursor := uuid.New()
	calls := make(chan uuid.UUID, 2)
	uc := &MockUsecase{
		RecoverPendingFunc: func(_ context.Context, after uuid.UUID) (tasksRecover.Output, error) {
			calls <- after
			if after == uuid.Nil {
				return tasksRecover.Output{NextRunID: cursor}, errors.New("one attempt failed")
			}
			return tasksRecover.Output{}, nil
		},
	}
	worker, err := NewWorker(uc, time.Hour, zerolog.Nop())
	require.NoError(t, err)
	ctx, cancel := context.WithTimeout(t.Context(), 5*time.Second)
	defer cancel()
	done := make(chan error, 1)
	go func() { done <- worker.Run(ctx) }()
	for _, expected := range []uuid.UUID{uuid.Nil, cursor} {
		select {
		case actual := <-calls:
			require.Equal(t, expected, actual)
		case <-ctx.Done():
			t.Fatal("worker waited instead of reading the next page")
		}
	}
	cancel()
	require.ErrorIs(t, <-done, context.Canceled)
	require.Len(t, uc.RecoverPendingCalls(), 2)
}

func TestWorkerRetriesListingCursorAndRepeatsSweep(t *testing.T) {
	cursor := uuid.New()
	entered := make(chan uuid.UUID, 4)
	call := 0
	uc := &MockUsecase{
		RecoverPendingFunc: func(ctx context.Context, after uuid.UUID) (tasksRecover.Output, error) {
			entered <- after
			call++
			switch call {
			case 1:
				return tasksRecover.Output{NextRunID: cursor}, nil
			case 2:
				return tasksRecover.Output{NextRunID: cursor}, errors.New("database unavailable")
			case 3:
				return tasksRecover.Output{}, nil
			default:
				<-ctx.Done()
				return tasksRecover.Output{}, ctx.Err()
			}
		},
	}
	worker, err := NewWorker(uc, time.Millisecond, zerolog.Nop())
	require.NoError(t, err)
	ctx, cancel := context.WithTimeout(t.Context(), 5*time.Second)
	defer cancel()
	done := make(chan error, 1)
	go func() { done <- worker.Run(ctx) }()
	for _, expected := range []uuid.UUID{uuid.Nil, cursor, cursor, uuid.Nil} {
		select {
		case actual := <-entered:
			require.Equal(t, expected, actual)
		case <-ctx.Done():
			t.Fatal("worker did not retry/repeat the sweep")
		}
	}
	cancel()
	require.ErrorIs(t, <-done, context.Canceled)
}

func TestWorkerRejectsInvalidConfiguration(t *testing.T) {
	_, err := NewWorker(nil, time.Second, zerolog.Nop())
	require.ErrorIs(t, err, errs.ErrInvalidArgument)
	_, err = NewWorker(&MockUsecase{}, 0, zerolog.Nop())
	require.ErrorIs(t, err, errs.ErrInvalidArgument)
}
