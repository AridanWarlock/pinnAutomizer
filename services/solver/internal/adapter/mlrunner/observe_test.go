package mlrunner

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/AridanWarlock/pinnAutomizer/pkg/docker"
	"github.com/AridanWarlock/pinnAutomizer/solver/internal/domain"
	"github.com/AridanWarlock/pinnAutomizer/solver/internal/domain/fixtures"
)

func TestObserveAttemptReadsResultWithoutChangingOriginal(t *testing.T) {
	t.Parallel()
	for _, code := range []int{0, 7} {
		a := fixtures.NewAttempt(t, fixtures.WithContainer("container-id"), fixtures.WithStartRequested())
		original := a
		c := ownedContainer(a)
		c.State = "exited"
		c.StartedAt = time.Now()
		c.ExitCode = code
		client := &MockDocker{
			InspectFunc: func(context.Context, string) (docker.Container, error) { return c, nil },
		}
		observed, err := (&PinnRunner{docker: client, pinnImage: "test-image"}).ObserveAttempt(t.Context(), a)
		require.NoError(t, err)
		require.True(t, observed.IsFinished())
		require.Equal(t, code, *observed.ExitCode)
		if code == 0 {
			require.Nil(t, observed.Error)
		} else {
			require.Equal(t, "container exited with code 7", *observed.Error)
		}
		require.Equal(t, original, a)
		require.Empty(t, client.StartCalls())
		require.Empty(t, client.CreateCalls())
	}
}

func TestObserveAttemptDoesNotInventCompletion(t *testing.T) {
	t.Parallel()
	for _, state := range []string{"running", "created", "dead", "canceled"} {
		t.Run(state, func(t *testing.T) {
			a := fixtures.NewAttempt(t, fixtures.WithContainer("container-id"), fixtures.WithStartRequested())
			c := ownedContainer(a)
			c.State = state
			c.Running = state == "running"
			client := &MockDocker{
				InspectFunc: func(context.Context, string) (docker.Container, error) {
					if state == "canceled" {
						return docker.Container{}, context.Canceled
					}
					return c, nil
				},
			}
			observed, err := (&PinnRunner{docker: client, pinnImage: "test-image"}).ObserveAttempt(t.Context(), a)
			switch state {
			case "running":
				require.NoError(t, err)
				require.Equal(t, a, observed)
			case "canceled":
				require.ErrorIs(t, err, context.Canceled)
			default:
				require.ErrorIs(t, err, domain.ErrStartOutcomeUnknown)
			}
			require.False(t, observed.IsFinished())
			require.Empty(t, client.StartCalls())
		})
	}
}
