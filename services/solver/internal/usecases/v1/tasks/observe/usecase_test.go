package tasksObserve

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"

	"github.com/AridanWarlock/pinnAutomizer/pkg/core"
	"github.com/AridanWarlock/pinnAutomizer/solver/internal/domain"
	"github.com/AridanWarlock/pinnAutomizer/solver/internal/domain/fixtures"
)

func completed(t *testing.T, a domain.Attempt) domain.Attempt {
	t.Helper()
	code := 0
	_, err := a.Finish(&code, nil, time.Now())
	require.NoError(t, err)
	return a
}

func TestObserveTaskCommitsOnlyObservedCompletion(t *testing.T) {
	t.Parallel()
	for _, scenario := range []string{"success", "pending", "already finished", "Docker error", "commit error", "another caller finished"} {
		t.Run(scenario, func(t *testing.T) {
			a := fixtures.NewAttempt(t, fixtures.WithContainer("container-id"), fixtures.WithStartRequested())
			if scenario == "already finished" {
				a = completed(t, a)
			}
			failure := errors.New("injected failure")
			sqlite := &MockSQLite{
				GetAttemptFunc: func(context.Context, uuid.UUID) (domain.Attempt, error) { return a, nil },
				FinishAttemptFunc: func(_ context.Context, id uuid.UUID, code *int, failureText *string, now time.Time, event domain.Event) (core.Changed, error) {
					require.Equal(t, a.RunID, id)
					require.Zero(t, *code)
					require.Nil(t, failureText)
					require.False(t, now.IsZero())
					require.Equal(t, id, event.RunID)
					if scenario == "commit error" {
						return false, failure
					}
					return core.Changed(scenario != "another caller finished"), nil
				},
			}
			runner := &MockPinnRunner{
				ObserveAttemptFunc: func(_ context.Context, a domain.Attempt) (domain.Attempt, error) {
					if scenario == "Docker error" {
						return domain.Attempt{}, failure
					}
					if scenario == "pending" {
						return a, nil
					}
					return completed(t, a), nil
				},
			}
			out, err := New(sqlite, runner).ObserveTask(t.Context(), Input{RunID: a.RunID})
			if scenario == "Docker error" || scenario == "commit error" {
				require.ErrorIs(t, err, failure)
			} else {
				require.NoError(t, err)
			}
			require.Equal(t, core.Changed(scenario == "success"), out.Finished)
			if scenario == "pending" || scenario == "already finished" || scenario == "Docker error" {
				require.Empty(t, sqlite.FinishAttemptCalls())
			}
			if scenario == "already finished" {
				require.Empty(t, runner.ObserveAttemptCalls())
			}
		})
	}
}

func TestCompletionEventIdentityAndWireContract(t *testing.T) {
	t.Parallel()
	a := completed(t, fixtures.NewAttempt(t))
	first, err := newCompletionEvent(a)
	require.NoError(t, err)
	second, err := newCompletionEvent(a)
	require.NoError(t, err)
	require.Equal(t, first.ID, second.ID)
	require.Equal(t, first.ID.String(), first.Message.Headers[core.KafkaIdempotencyHeader])
	require.Equal(t, "tasks.after.run", first.Message.Topic)
	require.Equal(t, []byte(a.TaskID.String()), first.Message.Key)
	require.JSONEq(t, `{"task_id":"`+a.TaskID.String()+`"}`, string(first.Message.Value))
	other, err := newCompletionEvent(completed(t, fixtures.NewAttempt(t)))
	require.NoError(t, err)
	require.NotEqual(t, first.ID, other.ID)
}

func TestCompletionEventCarriesFailure(t *testing.T) {
	t.Parallel()
	a := fixtures.NewAttempt(t)
	code := 7
	failure := "container exited with code 7"
	_, err := a.Finish(&code, &failure, time.Now())
	require.NoError(t, err)
	event, err := newCompletionEvent(a)
	require.NoError(t, err)
	require.JSONEq(t, `{"task_id":"`+a.TaskID.String()+`","error":"container exited with code 7"}`, string(event.Message.Value))
}
