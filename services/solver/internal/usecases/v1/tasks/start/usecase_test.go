package tasksStart

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

func TestStartTaskPersistsIntentBeforeDockerStart(t *testing.T) {
	t.Parallel()
	for _, failureAt := range []string{"none", "ensure", "record", "intent", "reload", "start"} {
		t.Run(failureAt, func(t *testing.T) {
			a := fixtures.NewAttempt(t)
			failure := errors.New("injected failure")
			fail := func(stage string) error {
				if stage == failureAt {
					return failure
				}
				return nil
			}
			var steps []string
			sqlite := &MockSQLite{
				GetAttemptFunc: func(context.Context, uuid.UUID) (domain.Attempt, error) {
					if a.StartAlreadyHandled() {
						return a, fail("reload")
					}
					return a, nil
				},
				RecordContainerFunc: func(_ context.Context, id uuid.UUID, containerID string) (core.Changed, error) {
					steps = append(steps, "record")
					if err := fail("record"); err != nil {
						return false, err
					}
					require.Equal(t, a.RunID, id)
					return a.RecordContainer(containerID)
				},
				RequestStartFunc: func(_ context.Context, id uuid.UUID, now time.Time, timeout time.Duration) (core.Changed, error) {
					steps = append(steps, "intent")
					if err := fail("intent"); err != nil {
						return false, err
					}
					require.Equal(t, a.RunID, id)
					return a.RequestStart(now, timeout)
				},
			}
			runner := &MockPinnRunner{
				EnsureContainerFunc: func(context.Context, domain.Attempt) (string, error) {
					steps = append(steps, "ensure")
					return "container", fail("ensure")
				},
				StartContainerFunc: func(_ context.Context, stored domain.Attempt) error {
					steps = append(steps, "start")
					require.Equal(t, domain.AttemptStartRequested, stored.State)
					require.Equal(t, "container", stored.ContainerID)
					require.NotNil(t, stored.Deadline)
					return fail("start")
				},
			}
			uc := New(sqlite, runner, time.Minute)
			out, err := uc.StartTask(t.Context(), Input{RunID: a.RunID})
			if failureAt == "none" {
				require.NoError(t, err)
				require.Equal(t, []string{"ensure", "record", "intent", "start"}, steps)
			} else {
				require.ErrorIs(t, err, failure)
			}
			require.Equal(t, core.Changed(a.StartAlreadyHandled()), out.StartRequested)
			if a.StartAlreadyHandled() {
				before := len(steps)
				_, err = uc.StartTask(t.Context(), Input{RunID: a.RunID})
				if failureAt == "reload" {
					require.ErrorIs(t, err, failure)
				} else {
					require.NoError(t, err)
				}
				require.Len(t, steps, before, "a consumed start intent must never send Start again")
			} else {
				require.Empty(t, runner.StartContainerCalls())
			}
		})
	}
}

func TestStartTaskSkipsDockerWhenAnotherCallerClaimsStart(t *testing.T) {
	t.Parallel()
	a := fixtures.NewAttempt(t)
	_, err := a.RecordContainer("container")
	require.NoError(t, err)
	sqlite := &MockSQLite{
		GetAttemptFunc:   func(context.Context, uuid.UUID) (domain.Attempt, error) { return a, nil },
		RequestStartFunc: func(context.Context, uuid.UUID, time.Time, time.Duration) (core.Changed, error) { return false, nil },
	}
	runner := &MockPinnRunner{}
	out, err := New(sqlite, runner, time.Minute).StartTask(t.Context(), Input{RunID: a.RunID})
	require.NoError(t, err)
	require.Equal(t, core.Changed(false), out.StartRequested)
	require.Empty(t, runner.EnsureContainerCalls())
	require.Empty(t, runner.StartContainerCalls())
}
