package tasksRecover

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"

	"github.com/AridanWarlock/pinnAutomizer/pkg/core/limit"
	"github.com/AridanWarlock/pinnAutomizer/pkg/errs"
	"github.com/AridanWarlock/pinnAutomizer/solver/internal/domain"
	"github.com/AridanWarlock/pinnAutomizer/solver/internal/domain/fixtures"
	tasksObserve "github.com/AridanWarlock/pinnAutomizer/solver/internal/usecases/v1/tasks/observe"
	tasksStart "github.com/AridanWarlock/pinnAutomizer/solver/internal/usecases/v1/tasks/start"
)

func TestRecoveryStartsOnlyBeforeDurableIntent(t *testing.T) {
	t.Parallel()
	for _, scenario := range []string{"accepted", "container created", "start requested", "start failure"} {
		t.Run(scenario, func(t *testing.T) {
			a := fixtures.NewAttempt(t)
			if scenario == "container created" || scenario == "start requested" {
				_, err := a.RecordContainer("container")
				require.NoError(t, err)
			}
			if scenario == "start requested" {
				_, err := a.RequestStart(time.Now(), time.Minute)
				require.NoError(t, err)
			}
			failure := errors.New("start unavailable")
			sqlite := &MockSQLite{
				GetUnfinishedAttemptsFunc: func(context.Context, uuid.UUID, limit.Limit) ([]domain.Attempt, error) {
					return []domain.Attempt{a}, nil
				},
			}
			start := &MockTasksStart{
				StartTaskFunc: func(_ context.Context, in tasksStart.Input) (tasksStart.Output, error) {
					require.Equal(t, a.RunID, in.RunID)
					if scenario == "start failure" {
						return tasksStart.Output{}, failure
					}
					return tasksStart.Output{}, nil
				},
			}
			observe := &MockTasksObserve{
				ObserveTaskFunc: func(_ context.Context, in tasksObserve.Input) (tasksObserve.Output, error) {
					require.Equal(t, a.RunID, in.RunID)
					return tasksObserve.Output{}, nil
				},
			}
			uc, err := New(sqlite, start, observe, limit.MustNew(2), time.Second)
			require.NoError(t, err)
			out, err := uc.RecoverPending(t.Context(), uuid.Nil)
			require.Equal(t, a.RunID, out.NextRunID)
			if scenario == "start failure" {
				require.ErrorIs(t, err, failure)
				require.Empty(t, observe.ObserveTaskCalls())
			} else {
				require.NoError(t, err)
				require.Len(t, observe.ObserveTaskCalls(), 1)
			}
			if scenario == "start requested" {
				require.Empty(t, start.StartTaskCalls())
			}
		})
	}
}

func TestRecoveryAdvancesPastUnknownStartAndTimedOutAttempt(t *testing.T) {
	t.Parallel()
	for _, timeout := range []bool{false, true} {
		a := fixtures.NewAttempt(t, fixtures.WithContainer("a"), fixtures.WithStartRequested())
		b := fixtures.NewAttempt(t, fixtures.WithContainer("b"), fixtures.WithStartRequested())
		sqlite := &MockSQLite{
			GetUnfinishedAttemptsFunc: func(_ context.Context, after uuid.UUID, batch limit.Limit) ([]domain.Attempt, error) {
				require.Equal(t, uint64(2), batch.Uint64())
				if after == uuid.Nil {
					return []domain.Attempt{a, b}, nil
				}
				require.Equal(t, b.RunID, after)
				return nil, nil
			},
		}
		var observed []uuid.UUID
		var firstDeadline time.Time
		observe := &MockTasksObserve{
			ObserveTaskFunc: func(ctx context.Context, in tasksObserve.Input) (tasksObserve.Output, error) {
				observed = append(observed, in.RunID)
				deadline, ok := ctx.Deadline()
				require.True(t, ok)
				if in.RunID == a.RunID {
					firstDeadline = deadline
					if timeout {
						<-ctx.Done()
						return tasksObserve.Output{}, ctx.Err()
					}
					return tasksObserve.Output{}, domain.ErrStartOutcomeUnknown
				}
				if timeout {
					require.True(t, deadline.After(firstDeadline))
				}
				return tasksObserve.Output{}, ctx.Err()
			},
		}
		attemptTimeout := time.Second
		if timeout {
			attemptTimeout = time.Millisecond
		}
		start := &MockTasksStart{}
		uc, err := New(sqlite, start, observe, limit.MustNew(2), attemptTimeout)
		require.NoError(t, err)
		out, err := uc.RecoverPending(t.Context(), uuid.Nil)
		if timeout {
			require.ErrorIs(t, err, context.DeadlineExceeded)
		} else {
			require.ErrorIs(t, err, domain.ErrStartOutcomeUnknown)
		}
		require.NoError(t, t.Context().Err())
		require.Equal(t, b.RunID, out.NextRunID)
		require.Equal(t, []uuid.UUID{a.RunID, b.RunID}, observed)
		require.Empty(t, start.StartTaskCalls())
		out, err = uc.RecoverPending(t.Context(), out.NextRunID)
		require.NoError(t, err)
		require.Equal(t, uuid.Nil, out.NextRunID)
	}
}

func TestRecoveryRetainsCursorOnReadFailureAndHonorsCancellation(t *testing.T) {
	t.Parallel()
	cursor := uuid.New()
	failure := errors.New("database unavailable")
	sqlite := &MockSQLite{
		GetUnfinishedAttemptsFunc: func(context.Context, uuid.UUID, limit.Limit) ([]domain.Attempt, error) { return nil, failure },
	}
	uc, err := New(sqlite, &MockTasksStart{}, &MockTasksObserve{}, limit.MustNew(1), time.Second)
	require.NoError(t, err)
	out, err := uc.RecoverPending(t.Context(), cursor)
	require.ErrorIs(t, err, failure)
	require.Equal(t, cursor, out.NextRunID)
	ctx, cancel := context.WithCancel(t.Context())
	cancel()
	_, err = uc.RecoverPending(ctx, cursor)
	require.ErrorIs(t, err, context.Canceled)
	require.Len(t, sqlite.GetUnfinishedAttemptsCalls(), 1)
}

func TestRecoveryRejectsInvalidConfiguration(t *testing.T) {
	t.Parallel()
	sqlite, start, observe := &MockSQLite{}, &MockTasksStart{}, &MockTasksObserve{}
	for _, timeout := range []time.Duration{0, -time.Second} {
		_, err := New(sqlite, start, observe, limit.MustNew(1), timeout)
		require.ErrorIs(t, err, errs.ErrInvalidArgument)
	}
	_, err := New(nil, start, observe, limit.MustNew(1), time.Second)
	require.ErrorIs(t, err, errs.ErrInvalidArgument)
	_, err = New(sqlite, nil, observe, limit.MustNew(1), time.Second)
	require.ErrorIs(t, err, errs.ErrInvalidArgument)
	_, err = New(sqlite, start, nil, limit.MustNew(1), time.Second)
	require.ErrorIs(t, err, errs.ErrInvalidArgument)
}

func TestRecoveryStopsBetweenAttemptsOnParentCancellation(t *testing.T) {
	t.Parallel()
	a := fixtures.NewAttempt(t, fixtures.WithContainer("a"), fixtures.WithStartRequested())
	b := fixtures.NewAttempt(t, fixtures.WithContainer("b"), fixtures.WithStartRequested())
	ctx, cancel := context.WithCancel(t.Context())
	defer cancel()
	sqlite := &MockSQLite{
		GetUnfinishedAttemptsFunc: func(context.Context, uuid.UUID, limit.Limit) ([]domain.Attempt, error) {
			return []domain.Attempt{a, b}, nil
		},
	}
	observe := &MockTasksObserve{
		ObserveTaskFunc: func(ctx context.Context, in tasksObserve.Input) (tasksObserve.Output, error) {
			cancel()
			return tasksObserve.Output{}, ctx.Err()
		},
	}
	uc, err := New(sqlite, &MockTasksStart{}, observe, limit.MustNew(2), time.Second)
	require.NoError(t, err)
	out, err := uc.RecoverPending(ctx, uuid.Nil)
	require.ErrorIs(t, err, context.Canceled)
	require.Equal(t, a.RunID, out.NextRunID)
	require.Len(t, observe.ObserveTaskCalls(), 1)
}
