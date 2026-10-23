package tasksObserve

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/rs/zerolog"
	"github.com/stretchr/testify/require"

	"github.com/AridanWarlock/pinnAutomizer/pkg/core"
	"github.com/AridanWarlock/pinnAutomizer/pkg/core/limit"
	"github.com/AridanWarlock/pinnAutomizer/pkg/docker"
	"github.com/AridanWarlock/pinnAutomizer/solver/internal/adapter/mlrunner"
	"github.com/AridanWarlock/pinnAutomizer/solver/internal/adapter/sqlite"
	"github.com/AridanWarlock/pinnAutomizer/solver/internal/domain/fixtures"
)

func TestDeadlineStopsDockerBeforePersistingResultIntegration(t *testing.T) {
	if os.Getenv("PINN_TEST_DOCKER") != "1" {
		t.Skip("set PINN_TEST_DOCKER=1 to test an overdue disposable CPU container")
	}
	ctx, cancel := context.WithTimeout(t.Context(), 40*time.Second)
	defer cancel()
	cfg, db := journal(t)
	repo, err := sqlite.New(cfg)
	require.NoError(t, err)
	defer repo.Close()
	a := fixtures.NewAttempt(t, fixtures.WithTime(time.Now().Add(-2*time.Minute)))
	_, _, err = repo.RegisterAttempt(ctx, a)
	require.NoError(t, err)
	cli, err := docker.New()
	require.NoError(t, err)
	defer cli.Close()
	name := "pinn-deadline-test-" + a.RunID.String()
	defer func() {
		cleanup, stop := context.WithTimeout(context.Background(), 10*time.Second)
		defer stop()
		require.NoError(t, cli.Remove(cleanup, name))
	}()
	image := os.Getenv("PINN_TEST_DOCKER_IMAGE")
	if image == "" {
		image = "golang:1.26.1-alpine"
	}
	id, err := cli.Create(ctx, docker.Config{
		Name:    name,
		Image:   image,
		Command: []string{"sh", "-c", "trap 'exit 0' TERM; while :; do sleep 1; done"},
		Labels: map[string]string{
			"pinn.solver.run_id":  a.RunID.String(),
			"pinn.solver.task_id": a.TaskID.String(),
			"pinn.solver.mode":    a.Mode,
		},
		NetworkMode:    "none",
		ReadonlyRootfs: true,
	})
	require.NoError(t, err)
	_, err = repo.RecordContainer(ctx, a.RunID, id)
	require.NoError(t, err)
	_, err = repo.RequestStart(ctx, a.RunID, a.CreatedAt, time.Minute, fixtures.NewEvent(t, a), limit.MustNew(1))
	require.NoError(t, err)
	// Simulate an overdue container left by an interrupted solver.
	require.NoError(t, cli.Start(ctx, id))
	before, err := cli.Inspect(ctx, id)
	require.NoError(t, err)
	require.True(t, before.Running)
	runner, err := mlrunner.NewPinnRunner(mlrunner.Config{
		ContainerNamePrefix: "pinn-deadline-test",
		Image:               image,
	}, zerolog.Nop())
	require.NoError(t, err)
	defer runner.Close()
	uc := New(repo, runner)
	_, err = db.Exec("CREATE TRIGGER reject_event BEFORE INSERT ON outbox BEGIN SELECT RAISE(ABORT,'injected failure'); END;")
	require.NoError(t, err)
	_, err = uc.ObserveTask(ctx, Input{RunID: a.RunID})
	require.Error(t, err)
	stopped, err := cli.Inspect(ctx, id)
	require.NoError(t, err)
	require.True(t, stopped.IsExited())
	require.False(t, stopped.Running)
	stored, err := repo.GetAttempt(ctx, a.RunID)
	require.NoError(t, err)
	require.False(t, stored.IsFinished(), "outbox failure must roll back the recorded result")
	_, err = db.Exec("DROP TRIGGER reject_event")
	require.NoError(t, err)
	out, err := uc.ObserveTask(ctx, Input{RunID: a.RunID})
	require.NoError(t, err)
	require.Equal(t, core.Changed(true), out.Finished)
	stored, err = repo.GetAttempt(ctx, a.RunID)
	require.NoError(t, err)
	require.Equal(t, "execution deadline exceeded", *stored.Error)
	events, err := repo.GetPendingEvents(ctx, limit.MustNew(10))
	require.NoError(t, err)
	require.Len(t, events, 2)
	require.Contains(t, string(events[1].Message.Value), "execution deadline exceeded")
	after, err := cli.Inspect(ctx, id)
	require.NoError(t, err)
	require.Equal(t, before.StartedAt, after.StartedAt)
	// The production path retains the container after commit; test cleanup owns removal.
	require.Equal(t, id, after.ID)
}
