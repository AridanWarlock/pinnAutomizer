package recovery

import (
	"context"
	"errors"
	"testing"
	"testing/synctest"
	"time"

	"github.com/google/uuid"
	"github.com/rs/zerolog"
	"github.com/stretchr/testify/require"

	"github.com/AridanWarlock/pinnAutomizer/pkg/errs"
	tasksRecover "github.com/AridanWarlock/pinnAutomizer/solver/internal/usecases/v1/tasks/recover"
)

func TestWorkerWaitsAfterEachBatchEvenWhenCursorAdvances(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		cursor := uuid.New()
		uc := &MockUsecase{
			RecoverPendingFunc: func(_ context.Context, in tasksRecover.Input) (tasksRecover.Output, error) {
				if in.AfterRunID == uuid.Nil {
					return tasksRecover.Output{NextRunID: cursor}, errors.New("one attempt failed")
				}
				return tasksRecover.Output{}, nil
			},
		}
		// synctest advances virtual time; this hour does not delay the test.
		const interval = time.Hour
		worker, err := NewWorker(uc, interval, zerolog.Nop())
		require.NoError(t, err)
		ctx, cancel := context.WithCancel(t.Context())
		defer cancel()
		done := make(chan error, 1)
		go func() { done <- worker.Run(ctx) }()
		synctest.Wait()
		require.Len(t, uc.RecoverPendingCalls(), 1, "only the first batch is immediate")
		<-time.After(interval - time.Nanosecond)
		synctest.Wait()
		require.Len(t, uc.RecoverPendingCalls(), 1)
		<-time.After(time.Nanosecond)
		synctest.Wait()
		calls := uc.RecoverPendingCalls()
		require.Len(t, calls, 2)
		require.Equal(t, cursor, calls[1].In.AfterRunID)
		cancel()
		require.ErrorIs(t, <-done, context.Canceled)
	})
}

func TestWorkerRetriesListingCursorAndRepeatsSweep(t *testing.T) {
	cursor := uuid.New()
	entered := make(chan uuid.UUID, 4)
	call := 0
	uc := &MockUsecase{
		RecoverPendingFunc: func(ctx context.Context, in tasksRecover.Input) (tasksRecover.Output, error) {
			after := in.AfterRunID
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
	for _, interval := range []time.Duration{0, -time.Second} {
		_, err := NewWorker(&MockUsecase{}, interval, zerolog.Nop())
		require.ErrorIs(t, err, errs.ErrInvalidArgument)
	}
}
