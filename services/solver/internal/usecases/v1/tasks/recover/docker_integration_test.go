package tasksRecover

import (
	"context"
	"database/sql"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/pressly/goose/v3"
	"github.com/rs/zerolog"
	"github.com/stretchr/testify/require"

	"github.com/AridanWarlock/pinnAutomizer/pkg/core/limit"
	"github.com/AridanWarlock/pinnAutomizer/pkg/docker"
	"github.com/AridanWarlock/pinnAutomizer/solver/internal/adapter/mlrunner"
	"github.com/AridanWarlock/pinnAutomizer/solver/internal/adapter/sqlite"
	"github.com/AridanWarlock/pinnAutomizer/solver/internal/domain"
	"github.com/AridanWarlock/pinnAutomizer/solver/internal/domain/fixtures"
	tasksCleanup "github.com/AridanWarlock/pinnAutomizer/solver/internal/usecases/v1/tasks/cleanup"
	tasksObserve "github.com/AridanWarlock/pinnAutomizer/solver/internal/usecases/v1/tasks/observe"
	tasksStart "github.com/AridanWarlock/pinnAutomizer/solver/internal/usecases/v1/tasks/start"
)

func recoveryJournal(t *testing.T) sqlite.Config {
	t.Helper()
	dir, err := filepath.Abs("../../../../../migrations")
	require.NoError(t, err)
	cfg := sqlite.Config{
		Path:          filepath.Join(t.TempDir(), "journal.db"),
		MigrationsDir: dir,
		BusyTimeout:   time.Second,
	}
	db, err := sql.Open("sqlite", cfg.Path)
	require.NoError(t, err)
	defer db.Close()
	provider, err := goose.NewProvider(goose.DialectSQLite3, db, os.DirFS(dir), goose.WithDisableGlobalRegistry(true))
	require.NoError(t, err)
	_, err = provider.Up(t.Context())
	require.NoError(t, err)
	return cfg
}

func TestRecoveryReopensIntentWithoutRestartingDockerIntegration(t *testing.T) {
	if os.Getenv("PINN_TEST_DOCKER") != "1" {
		t.Skip("set PINN_TEST_DOCKER=1 to test disposable CPU container recovery")
	}
	for _, executed := range []bool{false, true} {
		name := "intent before Start"
		if executed {
			name = "exited before persistence"
		}
		t.Run(name, func(t *testing.T) {
			ctx, cancel := context.WithTimeout(t.Context(), 30*time.Second)
			defer cancel()
			cfg := recoveryJournal(t)
			repo, err := sqlite.New(cfg)
			require.NoError(t, err)
			defer repo.Close()
			a := fixtures.NewAttempt(t)
			_, _, err = repo.RegisterAttempt(ctx, a)
			require.NoError(t, err)
			cli, err := docker.New()
			require.NoError(t, err)
			defer cli.Close()
			containerName := "pinn-recovery-test-" + a.RunID.String()
			defer func() {
				cleanup, stop := context.WithTimeout(context.Background(), 10*time.Second)
				defer stop()
				require.NoError(t, cli.Remove(cleanup, containerName))
			}()
			image := os.Getenv("PINN_TEST_DOCKER_IMAGE")
			if image == "" {
				image = "golang:1.26.1-alpine"
			}
			id, err := cli.Create(ctx, docker.Config{
				Name:    containerName,
				Image:   image,
				Command: []string{"sh", "-c", "echo recovered"},
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
			_, err = repo.RequestStart(ctx, a.RunID, time.Now(), time.Minute, fixtures.NewEvent(t, a), limit.MustNew(1))
			require.NoError(t, err)
			if executed {
				require.NoError(t, cli.Start(ctx, id))
				exit, err := cli.Wait(ctx, id)
				require.NoError(t, err)
				require.Zero(t, exit.Code)
			}
			before, err := cli.Inspect(ctx, id)
			require.NoError(t, err)
			// Recreate the repository/usecases after the external effect, without saving its result.
			require.NoError(t, repo.Close())
			repo, err = sqlite.New(cfg)
			require.NoError(t, err)
			defer repo.Close()
			runner, err := mlrunner.NewPinnRunner(mlrunner.Config{
				ContainerNamePrefix: "pinn-recovery-test",
				Image:               image,
			}, zerolog.Nop())
			require.NoError(t, err)
			defer runner.Close()
			uc, err := New(repo, tasksStart.New(repo, runner, time.Minute, limit.MustNew(1)), tasksObserve.New(repo, runner), tasksCleanup.New(repo, runner), limit.MustNew(2), 5*time.Second)
			require.NoError(t, err)
			out, err := uc.RecoverPending(ctx, Input{})
			require.Equal(t, a.RunID, out.NextRunID)
			if executed {
				require.NoError(t, err)
			} else {
				require.ErrorIs(t, err, domain.ErrStartOutcomeUnknown)
			}
			stored, err := repo.GetAttempt(ctx, a.RunID)
			require.NoError(t, err)
			require.Equal(t, executed, stored.IsFinished())
			events, err := repo.GetPendingEvents(ctx, limit.MustNew(10))
			require.NoError(t, err)
			if executed {
				require.Len(t, events, 2)
				require.Equal(t, "tasks.after.run", events[1].Message.Topic)
			} else {
				require.Len(t, events, 1, "uncertain Start keeps its intent event but must not produce completion")
				require.Equal(t, "fixture.start", events[0].Message.Topic)
			}
			after, err := cli.Inspect(ctx, id)
			require.NoError(t, err)
			require.Equal(t, before.StartedAt, after.StartedAt, "recovery must not restart a container")
			require.Equal(t, before.ID, after.ID)
		})
	}
}
