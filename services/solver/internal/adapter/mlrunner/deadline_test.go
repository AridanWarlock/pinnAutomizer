package mlrunner

import (
	"context"
	"errors"
	"testing"
	"testing/synctest"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/AridanWarlock/pinnAutomizer/pkg/docker"
	"github.com/AridanWarlock/pinnAutomizer/solver/internal/domain"
	"github.com/AridanWarlock/pinnAutomizer/solver/internal/domain/fixtures"
)

func TestExpiredRunningAttemptRequiresConfirmedStop(t *testing.T) {
	for _, scenario := range []string{"ack", "lost ack", "caller canceled", "still running", "inspect unavailable", "stop and inspect failed"} {
		t.Run(scenario, func(t *testing.T) {
			synctest.Test(t, func(t *testing.T) {
				a := fixtures.NewAttempt(t, fixtures.WithContainer("container-id"), fixtures.WithStartRequested())
				original := a
				c := ownedContainer(a)
				c.State = docker.StateRunning
				c.Running = true
				c.StartedAt = time.Now()
				<-time.After(time.Minute + time.Second)
				ctx, cancel := context.WithCancel(t.Context())
				defer cancel()
				client := &MockDocker{}
				client.InspectFunc = func(ctx context.Context, _ string) (docker.Container, error) {
					if len(client.StopCalls()) == 0 {
						if scenario == "caller canceled" {
							cancel()
						}
					} else {
						require.NoError(t, ctx.Err(), "confirmation uses an independent context")
						if scenario == "inspect unavailable" || scenario == "stop and inspect failed" {
							return docker.Container{}, errors.New("unavailable")
						}
					}
					return restoredContainer(t, c), nil
				}
				client.StopFunc = func(ctx context.Context, id string, grace time.Duration) error {
					require.NoError(t, ctx.Err())
					_, bounded := ctx.Deadline()
					require.True(t, bounded)
					require.Equal(t, a.ContainerID, id)
					require.Equal(t, stopGracePeriod, grace)
					if scenario != "still running" && scenario != "inspect unavailable" && scenario != "stop and inspect failed" {
						c.Running = false
						c.State = docker.StateExited
						c.Exit = &docker.Exit{}
						c.FinishedAt = time.Now()
					}
					if scenario == "lost ack" || scenario == "stop and inspect failed" {
						return errors.New("response lost")
					}
					return nil
				}
				result, err := (&PinnRunner{docker: client, pinnImage: "test-image"}).ObserveAttempt(ctx, a)
				if scenario == "still running" || (scenario == "inspect unavailable" || scenario == "stop and inspect failed") {
					require.ErrorIs(t, err, domain.ErrStopOutcomeUnknown)
					require.False(t, result.IsFinished())
				} else {
					require.NoError(t, err)
					require.True(t, result.IsFinished())
					require.Equal(t, "execution deadline exceeded", *result.Error)
				}
				require.Len(t, client.StopCalls(), 1)
				require.Len(t, client.InspectCalls(), 2, "reconcile Stop once, including a lost response")
				if scenario == "stop and inspect failed" {
					require.ErrorContains(t, err, "response lost")
					require.ErrorContains(t, err, "unavailable")
				}
				require.Empty(t, client.StartCalls())
				require.Equal(t, original, a)
			})
		})
	}
}

func TestObservedExitUsesFinishTimeForDeadline(t *testing.T) {
	for _, late := range []bool{false, true} {
		synctest.Test(t, func(t *testing.T) {
			a := fixtures.NewAttempt(t, fixtures.WithContainer("container-id"), fixtures.WithStartRequested())
			c := ownedContainer(a)
			c.State = docker.StateExited
			c.Exit = &docker.Exit{}
			c.StartedAt = time.Now()
			c.FinishedAt = a.Deadline.Add(-time.Nanosecond)
			if late {
				c.FinishedAt = a.Deadline.Add(time.Nanosecond)
			}
			<-time.After(2 * time.Minute)
			client := &MockDocker{
				InspectFunc: func(context.Context, string) (docker.Container, error) { return restoredContainer(t, c), nil },
			}
			result, err := (&PinnRunner{docker: client, pinnImage: "test-image"}).ObserveAttempt(t.Context(), a)
			require.NoError(t, err)
			require.True(t, result.IsFinished())
			if late {
				require.Equal(t, "execution deadline exceeded", *result.Error)
			} else {
				require.Nil(t, result.Error)
			}
			require.Empty(t, client.StopCalls())
		})
	}
}

func TestExpiredUncertainStartDoesNotInventStoppedResult(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		a := fixtures.NewAttempt(t, fixtures.WithContainer("container-id"), fixtures.WithStartRequested())
		c := ownedContainer(a)
		<-time.After(2 * time.Minute)
		client := &MockDocker{
			InspectFunc: func(context.Context, string) (docker.Container, error) { return restoredContainer(t, c), nil },
		}
		result, err := (&PinnRunner{docker: client, pinnImage: "test-image"}).ObserveAttempt(t.Context(), a)
		require.ErrorIs(t, err, domain.ErrStartOutcomeUnknown)
		require.False(t, result.IsFinished())
		require.Empty(t, client.StopCalls())
		require.Empty(t, client.StartCalls())
	})
}
