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
	tasksCleanup "github.com/AridanWarlock/pinnAutomizer/solver/internal/usecases/v1/tasks/cleanup"
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
				GetRecoverableAttemptsFunc: func(context.Context, uuid.UUID, limit.Limit) ([]domain.Attempt, error) {
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
			uc, err := New(sqlite, start, observe, &MockTasksCleanup{}, limit.MustNew(2), time.Second)
			require.NoError(t, err)
			out, err := uc.RecoverPending(t.Context(), Input{})
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
			GetRecoverableAttemptsFunc: func(_ context.Context, after uuid.UUID, batch limit.Limit) ([]domain.Attempt, error) {
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
		uc, err := New(sqlite, start, observe, &MockTasksCleanup{}, limit.MustNew(2), attemptTimeout)
		require.NoError(t, err)
		out, err := uc.RecoverPending(t.Context(), Input{})
		if timeout {
			require.ErrorIs(t, err, context.DeadlineExceeded)
		} else {
			require.ErrorIs(t, err, domain.ErrStartOutcomeUnknown)
		}
		require.NoError(t, t.Context().Err())
		require.Equal(t, b.RunID, out.NextRunID)
		require.Equal(t, []uuid.UUID{a.RunID, b.RunID}, observed)
		require.Empty(t, start.StartTaskCalls())
		out, err = uc.RecoverPending(t.Context(), Input{AfterRunID: out.NextRunID})
		require.NoError(t, err)
		require.Equal(t, uuid.Nil, out.NextRunID)
	}
}

func TestRecoveryRetainsCursorOnReadFailureAndHonorsCancellation(t *testing.T) {
	t.Parallel()
	cursor := uuid.New()
	failure := errors.New("database unavailable")
	sqlite := &MockSQLite{
		GetRecoverableAttemptsFunc: func(context.Context, uuid.UUID, limit.Limit) ([]domain.Attempt, error) { return nil, failure },
	}
	uc, err := New(sqlite, &MockTasksStart{}, &MockTasksObserve{}, &MockTasksCleanup{}, limit.MustNew(1), time.Second)
	require.NoError(t, err)
	out, err := uc.RecoverPending(t.Context(), Input{AfterRunID: cursor})
	require.ErrorIs(t, err, failure)
	require.Equal(t, cursor, out.NextRunID)
	ctx, cancel := context.WithCancel(t.Context())
	cancel()
	_, err = uc.RecoverPending(ctx, Input{AfterRunID: cursor})
	require.ErrorIs(t, err, context.Canceled)
	require.Len(t, sqlite.GetRecoverableAttemptsCalls(), 1)
}

func TestRecoveryRejectsInvalidConfiguration(t *testing.T) {
	t.Parallel()
	sqlite, start, observe := &MockSQLite{}, &MockTasksStart{}, &MockTasksObserve{}
	for _, timeout := range []time.Duration{0, -time.Second} {
		_, err := New(sqlite, start, observe, &MockTasksCleanup{}, limit.MustNew(1), timeout)
		require.ErrorIs(t, err, errs.ErrInvalidArgument)
	}

}

func TestRecoveryStopsBetweenAttemptsOnParentCancellation(t *testing.T) {
	t.Parallel()
	a := fixtures.NewAttempt(t, fixtures.WithContainer("a"), fixtures.WithStartRequested())
	b := fixtures.NewAttempt(t, fixtures.WithContainer("b"), fixtures.WithStartRequested())
	ctx, cancel := context.WithCancel(t.Context())
	defer cancel()
	sqlite := &MockSQLite{
		GetRecoverableAttemptsFunc: func(context.Context, uuid.UUID, limit.Limit) ([]domain.Attempt, error) {
			return []domain.Attempt{a, b}, nil
		},
	}
	observe := &MockTasksObserve{
		ObserveTaskFunc: func(ctx context.Context, in tasksObserve.Input) (tasksObserve.Output, error) {
			cancel()
			return tasksObserve.Output{}, ctx.Err()
		},
	}
	uc, err := New(sqlite, &MockTasksStart{}, observe, &MockTasksCleanup{}, limit.MustNew(2), time.Second)
	require.NoError(t, err)
	out, err := uc.RecoverPending(ctx, Input{})
	require.ErrorIs(t, err, context.Canceled)
	require.Equal(t, a.RunID, out.NextRunID)
	require.Len(t, observe.ObserveTaskCalls(), 1)
}

func TestRecoveryRoutesFinishedAttemptsOnlyToCleanupAndContinuesAfterFailure(t *testing.T) {
	t.Parallel()
	first := fixtures.NewAttempt(t, fixtures.WithContainer("first"))
	second := fixtures.NewAttempt(t, fixtures.WithContainer("second"))
	code := 0
	for _, a := range []*domain.Attempt{&first, &second} {
		_, err := a.Finish(&code, nil, time.Now())
		require.NoError(t, err)
	}
	sqlite := &MockSQLite{
		GetRecoverableAttemptsFunc: func(context.Context, uuid.UUID, limit.Limit) ([]domain.Attempt, error) {
			return []domain.Attempt{first, second}, nil
		},
	}
	failure := errors.New("remove unavailable")
	cleanup := &MockTasksCleanup{
		CleanupTaskFunc: func(ctx context.Context, in tasksCleanup.Input) (tasksCleanup.Output, error) {
			_, bounded := ctx.Deadline()
			require.True(t, bounded)
			if in.RunID == first.RunID {
				return tasksCleanup.Output{}, failure
			}
			require.Equal(t, second.RunID, in.RunID)
			return tasksCleanup.Output{}, nil
		},
	}
	start := &MockTasksStart{}
	observe := &MockTasksObserve{}
	uc, err := New(sqlite, start, observe, cleanup, limit.MustNew(2), time.Second)
	require.NoError(t, err)
	out, err := uc.RecoverPending(t.Context(), Input{})
	require.ErrorIs(t, err, failure)
	require.Equal(t, second.RunID, out.NextRunID, "cleanup failure must not starve the next attempt")
	require.Len(t, cleanup.CleanupTaskCalls(), 2)
	require.Empty(t, start.StartTaskCalls())
	require.Empty(t, observe.ObserveTaskCalls())
}
