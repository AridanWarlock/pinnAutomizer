package mlrunner

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/AridanWarlock/pinnAutomizer/pkg/docker"
	"github.com/AridanWarlock/pinnAutomizer/pkg/errs"
	"github.com/AridanWarlock/pinnAutomizer/solver/internal/domain/fixtures"
)

func TestCleanupOnlyRemovesOwnedFinishedContainers(t *testing.T) {
	t.Parallel()
	for _, scenario := range []string{"removed", "already absent", "lost remove response", "running", "foreign", "inspect failed", "remove failed"} {
		t.Run(scenario, func(t *testing.T) {
			a := fixtures.NewAttempt(t, fixtures.WithContainer("container-id"))
			code := 0
			_, err := a.Finish(&code, nil, time.Now())
			require.NoError(t, err)
			in := ownedContainer(a)
			in.State = docker.StateExited
			in.StartedAt = time.Now()
			in.FinishedAt = in.StartedAt
			in.Exit = &docker.Exit{}
			if scenario == "running" {
				in.State = docker.StateRunning
				in.Running = true
				in.Exit = nil
			}
			if scenario == "foreign" {
				in.Labels[runLabel] = "other"
			}
			failure := errors.New("docker unavailable")
			client := &MockDocker{
				InspectFunc: func(ctx context.Context, id string) (docker.Container, error) {
					require.NoError(t, ctx.Err())
					_, bounded := ctx.Deadline()
					require.True(t, bounded)
					require.Equal(t, a.ContainerID, id)
					if scenario == "already absent" {
						return docker.Container{}, errs.ErrNotFound
					}
					if scenario == "inspect failed" {
						return docker.Container{}, failure
					}
					return restoredContainer(t, in), nil
				},
				RemoveStoppedFunc: func(ctx context.Context, id string) error {
					require.NoError(t, ctx.Err())
					require.Equal(t, a.ContainerID, id)
					if scenario == "lost remove response" {
						return errs.ErrNotFound
					}
					if scenario == "remove failed" {
						return failure
					}
					return nil
				},
			}
			ctx := t.Context()
			err = (&PinnRunner{docker: client, pinnImage: "test-image"}).CleanupAttempt(ctx, a)
			switch scenario {
			case "foreign", "running":
				require.ErrorIs(t, err, errs.ErrConflict)
			case "inspect failed", "remove failed":
				require.ErrorIs(t, err, failure)
			default:
				require.NoError(t, err)
			}
			shouldRemove := scenario == "removed" || scenario == "lost remove response" || scenario == "remove failed"
			require.Equal(t, shouldRemove, len(client.RemoveStoppedCalls()) == 1)
			require.Empty(t, client.StartCalls())
			require.Empty(t, client.StopCalls())
		})
	}
}

func TestCleanupRefusesUnfinishedAttempt(t *testing.T) {
	t.Parallel()
	client := &MockDocker{}
	err := (&PinnRunner{docker: client}).CleanupAttempt(t.Context(), fixtures.NewAttempt(t))
	require.ErrorIs(t, err, errs.ErrConflict)
	require.Empty(t, client.InspectCalls())
	require.Empty(t, client.RemoveStoppedCalls())
}

func TestCleanupHonorsCallerCancellation(t *testing.T) {
	t.Parallel()
	a := fixtures.NewAttempt(t, fixtures.WithContainer("container-id"))
	code := 0
	_, err := a.Finish(&code, nil, time.Now())
	require.NoError(t, err)
	client := &MockDocker{
		InspectFunc: func(ctx context.Context, _ string) (docker.Container, error) {
			require.ErrorIs(t, ctx.Err(), context.Canceled)
			return docker.Container{}, ctx.Err()
		},
	}
	ctx, cancel := context.WithCancel(t.Context())
	cancel()
	err = (&PinnRunner{docker: client}).CleanupAttempt(ctx, a)
	require.ErrorIs(t, err, context.Canceled)
	require.Empty(t, client.RemoveStoppedCalls(), "canceled cleanup is deferred to a later recovery sweep")
}
