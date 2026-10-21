package docker

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/AridanWarlock/pinnAutomizer/pkg/errs"
)

func TestNewContainerRejectsInconsistentExecution(t *testing.T) {
	t.Parallel()
	now := time.Now()
	for _, scenario := range []struct {
		name  string
		input ContainerInput
	}{
		{
			name:  "missing identity",
			input: ContainerInput{State: StateCreated},
		},
		{
			name:  "unknown state",
			input: ContainerInput{ID: "id", State: "unknown"},
		},
		{
			name:  "running without start",
			input: ContainerInput{ID: "id", State: StateRunning, Running: true},
		},
		{
			name:  "running without active flag",
			input: ContainerInput{ID: "id", State: StateRunning, StartedAt: now},
		},
		{
			name:  "exit without start",
			input: ContainerInput{ID: "id", State: StateExited, FinishedAt: now},
		},
		{
			name:  "exit without finish",
			input: ContainerInput{ID: "id", State: StateExited, StartedAt: now},
		},
		{
			name:  "exit still running",
			input: ContainerInput{ID: "id", State: StateExited, Running: true, StartedAt: now, FinishedAt: now},
		},
		{
			name:  "finish precedes start",
			input: ContainerInput{ID: "id", State: StateExited, StartedAt: now, FinishedAt: now.Add(-time.Second)},
		},
	} {
		t.Run(scenario.name, func(t *testing.T) {
			_, err := NewContainer(scenario.input)
			require.ErrorIs(t, err, errs.ErrInvalidArgument, "input: %+v", scenario.input)
		})
	}
}

func TestContainerStatesDescribeConfirmedExecution(t *testing.T) {
	t.Parallel()
	now := time.Now()
	for _, state := range []State{StateCreated, StateRunning, StatePaused, StateRestarting, StateRemoving, StateExited, StateDead} {
		t.Run(string(state), func(t *testing.T) {
			in := ContainerInput{
				ID:     "id",
				State:  state,
				Labels: map[string]string{"owner": "solver"},
			}
			switch state {
			case StateRunning:
				in.Running = true
				in.StartedAt = now
			case StateExited:
				in.StartedAt = now
				in.FinishedAt = now
			}
			c, err := NewContainer(in)
			require.NoError(t, err)
			require.Equal(t, state == StateCreated, c.IsCreated())
			require.Equal(t, state == StateRunning, c.IsRunning())
			require.Equal(t, state == StateRunning || state == StateExited, c.HasStarted())
			require.Equal(t, state == StateExited, c.IsFinished())
			in.Labels["owner"] = "other"
			require.Equal(t, "solver", c.Labels["owner"], "constructor must own its labels")
		})
	}
}
