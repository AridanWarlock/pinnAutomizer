package tasksObserve

import (
	"context"
	"errors"
	"os"
	"slices"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/rs/zerolog"
	"github.com/stretchr/testify/require"

	"github.com/AridanWarlock/pinnAutomizer/pkg/core"
	"github.com/AridanWarlock/pinnAutomizer/pkg/core/limit"
	"github.com/AridanWarlock/pinnAutomizer/pkg/docker"
	"github.com/AridanWarlock/pinnAutomizer/pkg/errs"
	"github.com/AridanWarlock/pinnAutomizer/solver/internal/adapter/mlrunner"
	"github.com/AridanWarlock/pinnAutomizer/solver/internal/adapter/sqlite"
	"github.com/AridanWarlock/pinnAutomizer/solver/internal/domain"
	"github.com/AridanWarlock/pinnAutomizer/solver/internal/domain/fixtures"
	tasksCleanup "github.com/AridanWarlock/pinnAutomizer/solver/internal/usecases/v1/tasks/cleanup"
)

func TestDockerCleanupRetriesAfterJournalFailureAndReopenIntegration(t *testing.T) {
	if os.Getenv("PINN_TEST_DOCKER") != "1" {
		t.Skip("set PINN_TEST_DOCKER=1 to test disposable CPU container cleanup")
	}
	ctx, cancel := context.WithTimeout(t.Context(), 40*time.Second)
	defer cancel()
	cfg, db := journal(t)
	repo, err := sqlite.New(cfg)
	require.NoError(t, err)
	defer func() { require.NoError(t, repo.Close()) }()
	a := fixtures.NewAttempt(t)
	cli, err := docker.New()
	require.NoError(t, err)
	defer cli.Close()
	name := "pinn-cleanup-test-" + a.RunID.String()
	defer func() {
		cleanup, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		err := cli.Remove(cleanup, name)
		if !errors.Is(err, errs.ErrNotFound) {
			require.NoError(t, err)
		}
	}()
	image := os.Getenv("PINN_TEST_DOCKER_IMAGE")
	if image == "" {
		image = "golang:1.26.1-alpine"
	}
	_, _, err = repo.RegisterAttempt(ctx, a)
	require.NoError(t, err)
	id, err := cli.Create(ctx, docker.Config{
		Name:    name,
		Image:   image,
		Command: []string{"sh", "-c", "exit 0"},
		Labels: map[string]string{
			"pinn.solver.run_id":  a.RunID.String(),
			"pinn.solver.task_id": a.TaskID.String(),
			"pinn.solver.mode":    a.Mode,
		},
		NetworkMode: "none",
	})
	require.NoError(t, err)
	_, err = repo.RecordContainer(ctx, a.RunID, id)
	require.NoError(t, err)
	_, err = repo.RequestStart(ctx, a.RunID, time.Now(), time.Minute, fixtures.NewEvent(t, a))
	require.NoError(t, err)
	require.NoError(t, cli.Start(ctx, id))
	_, err = cli.Wait(ctx, id)
	require.NoError(t, err)
	runner, err := mlrunner.NewPinnRunner(mlrunner.Config{
		ContainerNamePrefix: "pinn-cleanup-test",
		Image:               image,
	}, zerolog.Nop())
	require.NoError(t, err)
	defer runner.Close()
	observed, err := New(repo, runner).ObserveTask(ctx, Input{RunID: a.RunID})
	require.NoError(t, err)
	require.Equal(t, core.Changed(true), observed.Finished)
	before, err := repo.GetAttempt(ctx, a.RunID)
	require.NoError(t, err)
	eventsBefore, err := repo.GetPendingEvents(ctx, limit.MustNew(10))
	require.NoError(t, err)
	_, err = db.Exec("CREATE TRIGGER reject_cleanup BEFORE UPDATE OF cleaned_at ON attempts WHEN NEW.cleaned_at IS NOT NULL BEGIN SELECT RAISE(ABORT,'injected cleanup failure'); END;")
	require.NoError(t, err)
	_, err = tasksCleanup.New(repo, runner).CleanupTask(ctx, tasksCleanup.Input{RunID: a.RunID})
	require.ErrorContains(t, err, "injected cleanup failure")
	_, err = cli.Inspect(ctx, id)
	require.ErrorIs(t, err, errs.ErrNotFound, "physical deletion must precede marker")
	pending, err := repo.GetRecoverableAttempts(ctx, uuid.Nil, limit.MustNew(10))
	require.NoError(t, err)
	require.True(t, slices.ContainsFunc(pending, func(p domain.Attempt) bool { return p.RunID == a.RunID }))
	require.NoError(t, repo.Close())
	repo, err = sqlite.New(cfg)
	require.NoError(t, err)
	_, err = db.Exec("DROP TRIGGER reject_cleanup")
	require.NoError(t, err)
	uc := tasksCleanup.New(repo, runner)
	out, err := uc.CleanupTask(ctx, tasksCleanup.Input{RunID: a.RunID})
	require.NoError(t, err)
	require.Equal(t, core.Changed(true), out.Cleaned)
	out, err = uc.CleanupTask(ctx, tasksCleanup.Input{RunID: a.RunID})
	require.NoError(t, err)
	require.Equal(t, core.Changed(false), out.Cleaned)
	after, err := repo.GetAttempt(ctx, a.RunID)
	require.NoError(t, err)
	require.NotNil(t, after.CleanedAt)
	require.Equal(t, before.CompletedAt, after.CompletedAt)
	require.Equal(t, before.ExitCode, after.ExitCode)
	require.Equal(t, before.Error, after.Error)
	eventsAfter, err := repo.GetPendingEvents(ctx, limit.MustNew(10))
	require.NoError(t, err)
	require.Equal(t, eventsBefore, eventsAfter, "cleanup must not recreate or change result events")
}
