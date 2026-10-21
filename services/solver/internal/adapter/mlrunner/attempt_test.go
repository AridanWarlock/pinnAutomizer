package mlrunner

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/AridanWarlock/pinnAutomizer/pkg/docker"
	"github.com/AridanWarlock/pinnAutomizer/pkg/errs"
	"github.com/AridanWarlock/pinnAutomizer/solver/internal/domain"
	"github.com/AridanWarlock/pinnAutomizer/solver/internal/domain/fixtures"
)

func ownedContainer(a domain.Attempt) docker.ContainerInput {
	return docker.ContainerInput{
		ID:    "container-id",
		Name:  "/" + (&PinnRunner{}).containerName(a.RunID),
		Image: "test-image",
		State: docker.StateCreated,
		Labels: map[string]string{
			runLabel:  a.RunID.String(),
			taskLabel: a.TaskID.String(),
			modeLabel: a.Mode,
		},
	}
}

func TestEnsureContainerReconcilesLostCreateResponse(t *testing.T) {
	t.Parallel()
	a := fixtures.NewAttempt(t)
	existing := ownedContainer(a)
	for _, scenario := range []struct {
		name          string
		cancelRequest bool
	}{
		{name: "lost response"},
		{name: "canceled response", cancelRequest: true},
	} {
		t.Run(scenario.name, func(t *testing.T) {
			ctx, cancel := context.WithCancel(t.Context())
			defer cancel()
			created := false
			client := &MockDocker{
				InspectFunc: func(ctx context.Context, id string) (docker.Container, error) {
					require.NoError(t, ctx.Err())
					require.Equal(t, existing.Name[1:], id)
					if !created {
						return docker.Container{}, errs.ErrNotFound
					}
					return restoredContainer(t, existing), nil
				},
				CreateFunc: func(_ context.Context, cfg docker.Config) (string, error) {
					require.Equal(t, existing.Name[1:], cfg.Name)
					require.False(t, cfg.AutoRemove)
					require.Equal(t, a.RunID.String(), cfg.Labels[runLabel])
					require.Equal(t, []string{"python", "run.py", "--mod=predict"}, cfg.Command)
					created = true
					if scenario.cancelRequest {
						cancel()
						return "", context.Canceled
					}
					return "", errors.New("lost response")
				},
			}
			runner := &PinnRunner{docker: client, pinnImage: "test-image"}
			id, err := runner.EnsureContainer(ctx, a)
			require.NoError(t, err)
			require.Equal(t, existing.ID, id)
			id, err = runner.EnsureContainer(t.Context(), a)
			require.NoError(t, err)
			require.Equal(t, existing.ID, id)
			require.Len(t, client.CreateCalls(), 1)
		})
	}
}

func TestEnsureContainerDoesNotCreateOnUnavailableInspect(t *testing.T) {
	t.Parallel()
	failure := errors.New("unavailable")
	client := &MockDocker{
		InspectFunc: func(context.Context, string) (docker.Container, error) { return docker.Container{}, failure },
	}
	_, err := (&PinnRunner{docker: client}).EnsureContainer(t.Context(), fixtures.NewAttempt(t))
	require.ErrorIs(t, err, failure)
	require.Empty(t, client.CreateCalls())
}

func TestEnsureContainerRejectsForeignOrUnsafeContainer(t *testing.T) {
	t.Parallel()
	for _, scenario := range []struct {
		name   string
		change func(*docker.ContainerInput)
	}{
		{"foreign run", func(c *docker.ContainerInput) { c.Labels[runLabel] = "other" }},
		{"different mode", func(c *docker.ContainerInput) { c.Labels[modeLabel] = "train" }},
		{"different image", func(c *docker.ContainerInput) { c.Image = "other" }},
		{"automatic removal", func(c *docker.ContainerInput) { c.AutoRemove = true }},
		{"automatic restart", func(c *docker.ContainerInput) { c.RestartPolicy = "always" }},
	} {
		t.Run(scenario.name, func(t *testing.T) {
			a := fixtures.NewAttempt(t)
			existing := ownedContainer(a)
			scenario.change(&existing)
			client := &MockDocker{
				InspectFunc: func(context.Context, string) (docker.Container, error) { return restoredContainer(t, existing), nil },
			}
			_, err := (&PinnRunner{docker: client, pinnImage: "test-image"}).EnsureContainer(t.Context(), a)
			require.ErrorIs(t, err, errs.ErrConflict)
			require.Empty(t, client.CreateCalls())
		})
	}
}

func TestStartContainerNeverRetriesStart(t *testing.T) {
	t.Parallel()
	for _, state := range []string{"running", "exited", "created", "unavailable"} {
		t.Run(state, func(t *testing.T) {
			a := fixtures.NewAttempt(t, fixtures.WithContainer("container-id"), fixtures.WithStartRequested())
			existing := ownedContainer(a)
			client := &MockDocker{}
			client.InspectFunc = func(context.Context, string) (docker.Container, error) {
				if len(client.StartCalls()) > 0 && state == "unavailable" {
					return docker.Container{}, errors.New("inspect unavailable")
				}
				return restoredContainer(t, existing), nil
			}
			client.StartFunc = func(context.Context, string) error {
				existing.State = docker.State(state)
				existing.Running = state == "running"
				if state == "running" || state == "exited" {
					existing.StartedAt = time.Now()
					if state == "exited" {
						existing.FinishedAt = time.Now()
					}
				}
				return errors.New("lost start response")
			}
			runner := &PinnRunner{docker: client, pinnImage: "test-image"}
			err := runner.StartContainer(t.Context(), a)
			if state == "created" || state == "unavailable" {
				require.ErrorIs(t, err, domain.ErrStartOutcomeUnknown)
			} else {
				require.NoError(t, err)
				require.NoError(t, runner.StartContainer(t.Context(), a))
			}
			require.Len(t, client.StartCalls(), 1)
		})
	}
}

func TestStartContainerReconcilesAfterCancellation(t *testing.T) {
	t.Parallel()
	a := fixtures.NewAttempt(t, fixtures.WithContainer("container-id"), fixtures.WithStartRequested())
	existing := ownedContainer(a)
	ctx, cancel := context.WithCancel(t.Context())
	defer cancel()
	client := &MockDocker{
		InspectFunc: func(ctx context.Context, _ string) (docker.Container, error) {
			require.NoError(t, ctx.Err())
			return restoredContainer(t, existing), nil
		},
		StartFunc: func(context.Context, string) error {
			cancel()
			existing.Running = true
			existing.State = docker.StateRunning
			existing.StartedAt = time.Now()
			return context.Canceled
		},
	}
	require.NoError(t, (&PinnRunner{docker: client, pinnImage: "test-image"}).StartContainer(ctx, a))
	require.Len(t, client.StartCalls(), 1)
}

func TestStartContainerRequiresUnexpiredIntent(t *testing.T) {
	t.Parallel()
	a := fixtures.NewAttempt(t)
	client := &MockDocker{}
	runner := &PinnRunner{docker: client}
	require.ErrorIs(t, runner.StartContainer(t.Context(), a), errs.ErrConflict)
	_, err := a.RecordContainer("container-id")
	require.NoError(t, err)
	_, err = a.RequestStart(time.Now().Add(-time.Hour), time.Minute)
	require.NoError(t, err)
	require.ErrorIs(t, runner.StartContainer(t.Context(), a), context.DeadlineExceeded)
	require.Empty(t, client.InspectCalls())
	require.Empty(t, client.StartCalls())
}

func restoredContainer(t *testing.T, in docker.ContainerInput) docker.Container {
	t.Helper()
	c, err := docker.NewContainer(in)
	require.NoError(t, err)
	return c
}
