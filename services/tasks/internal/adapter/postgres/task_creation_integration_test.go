package postgres

import (
	"context"
	"errors"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/rs/zerolog"
	"github.com/stretchr/testify/require"

	"github.com/AridanWarlock/pinnAutomizer/pkg/core/corefixtures"
	"github.com/AridanWarlock/pinnAutomizer/pkg/core/limit"
	"github.com/AridanWarlock/pinnAutomizer/pkg/errs"
	"github.com/AridanWarlock/pinnAutomizer/pkg/fileservice"
	"github.com/AridanWarlock/pinnAutomizer/tasks/internal/adapter/filestore"
	"github.com/AridanWarlock/pinnAutomizer/tasks/internal/domain"
	tasksCleanup "github.com/AridanWarlock/pinnAutomizer/tasks/internal/usecases/v1/tasks/cleanup"
	tasksCreate "github.com/AridanWarlock/pinnAutomizer/tasks/internal/usecases/v1/tasks/create"
)

// Tests mount logical /tasks_* paths inside a temporary volume.
type creationVolume struct {
	root string
}

func (v creationVolume) path(path string) string {
	return filepath.Join(v.root, strings.TrimPrefix(path, "/"))
}

func (v creationVolume) WithDirectoryLock(
	ctx context.Context,
	path string,
	process func(context.Context) error,
) error {
	return fileservice.NewFileService().WithDirectoryLock(ctx, v.path(path), process)
}

func (v creationVolume) MkdirAll(path string) error {
	return os.MkdirAll(v.path(path), 0755)
}

func (v creationVolume) RemoveAll(path string) error {
	return os.RemoveAll(v.path(path))
}

func (v creationVolume) Store(path string, files []domain.TaskFile) error {
	return filestore.NewFileStore(fileservice.NewFileService()).Store(v.path(path), files)
}

type observingCreationVolume struct {
	creationVolume
	hook func() error
}

func (v observingCreationVolume) Store(path string, files []domain.TaskFile) error {
	if err := v.creationVolume.Store(path, files); err != nil {
		return err
	}
	return v.hook()
}

type staleCleanupPostgres struct {
	*Repository
	jobs []domain.FileCleanupJob
}

func (p staleCleanupPostgres) GetFileCleanupJobs(context.Context, uuid.UUID, limit.Limit) ([]domain.FileCleanupJob, error) {
	return p.jobs, nil
}

type inspectedCreationPostgres struct {
	*Repository
	beforeConfirm func() error
}

func (p inspectedCreationPostgres) DeleteFileCleanupJob(ctx context.Context, taskID uuid.UUID) error {
	if err := p.beforeConfirm(); err != nil {
		return err
	}
	return p.Repository.DeleteFileCleanupJob(ctx, taskID)
}

func TestCreationStagingAndStaleCleanupIntegration(t *testing.T) {
	for _, failureAt := range []string{"none", "write", "confirm"} {
		t.Run(failureAt, func(t *testing.T) {
			repo, db := newTaskIntegrationRepository(t)
			ctx := corefixtures.NewAuthInfo().WithContext(t.Context())
			volume := creationVolume{root: t.TempDir()}
			failure := errors.New("partial upload")
			var staged []domain.FileCleanupJob
			files := observingCreationVolume{
				creationVolume: volume,
				hook: func() error {
					var active int
					err := db.QueryRow(ctx, "SELECT COUNT(*) FROM pg_stat_activity WHERE datname=current_database() AND state='idle in transaction'").Scan(&active)
					require.NoError(t, err)
					require.Zero(t, active, "file I/O must not hold a transaction")
					staged, err = repo.GetFileCleanupJobs(ctx, uuid.Nil, limit.MustNew(32))
					require.NoError(t, err)
					require.Len(t, staged, 1)
					_, err = repo.GetTaskByID(ctx, staged[0].TaskID)
					require.ErrorIs(t, err, errs.ErrNotFound, "unfinished task must not be visible")
					_, err = tasksCleanup.New(repo, volume, limit.MustNew(32)).CleanupPending(ctx, tasksCleanup.Input{})
					require.ErrorIs(t, err, errs.ErrConflict, "cleanup must not enter active upload")
					if failureAt == "write" {
						return failure
					}
					if failureAt == "confirm" {
						_, err = db.Exec(ctx, `CREATE FUNCTION reject_confirmation() RETURNS trigger LANGUAGE plpgsql AS $$
 BEGIN RAISE EXCEPTION 'confirmation failed'; END $$;
 CREATE TRIGGER reject_confirmation BEFORE DELETE ON file_cleanup FOR EACH ROW EXECUTE FUNCTION reject_confirmation();`)
						require.NoError(t, err)
					}
					return nil
				},
			}
			source, err := os.CreateTemp(t.TempDir(), "input")
			require.NoError(t, err)
			defer source.Close()
			_, err = source.WriteString("uploaded data")
			require.NoError(t, err)
			_, err = source.Seek(0, 0)
			require.NoError(t, err)
			creationPostgres := inspectedCreationPostgres{
				Repository: repo,
				beforeConfirm: func() error {
					// INSERT has executed but confirmation has not: cleanup still cannot enter.
					_, err := tasksCleanup.New(repo, volume, limit.MustNew(32)).CleanupPending(ctx, tasksCleanup.Input{})
					require.ErrorIs(t, err, errs.ErrConflict)
					require.FileExists(t, volume.path(filepath.Join(staged[0].DataPath, "data.txt")))
					return nil
				},
			}
			out, err := tasksCreate.New(creationPostgres, files).CreateTask(ctx, tasksCreate.Input{
				Name: "staged", Mode: domain.TaskModeTrain,
				Files: []domain.TaskFile{{Name: "data.txt", File: source}},
			})
			require.Len(t, staged, 1)
			job := staged[0]
			if failureAt == "none" {
				require.NoError(t, err)
				require.Equal(t, job.TaskID, out.Task.ID)
				// A worker selected this job before commit, but acquires its lock afterwards.
				_, err = tasksCleanup.New(staleCleanupPostgres{Repository: repo, jobs: staged}, volume, limit.MustNew(32)).CleanupPending(ctx, tasksCleanup.Input{})
				require.NoError(t, err)
				content, err := os.ReadFile(volume.path(filepath.Join(job.DataPath, "data.txt")))
				require.NoError(t, err)
				require.Equal(t, "uploaded data", string(content), "stale job must not delete committed files")
			} else {
				require.Error(t, err)
				_, getErr := repo.GetTaskByID(ctx, job.TaskID)
				require.ErrorIs(t, getErr, errs.ErrNotFound)
				pending, getErr := repo.HasFileCleanupJob(ctx, job.TaskID)
				require.NoError(t, getErr)
				require.True(t, pending, "rollback must retain durable cleanup")
				if failureAt == "confirm" {
					_, err = db.Exec(ctx, "DROP TRIGGER reject_confirmation ON file_cleanup")
					require.NoError(t, err)
				}
				_, err = tasksCleanup.New(repo, volume, limit.MustNew(32)).CleanupPending(ctx, tasksCleanup.Input{})
				require.NoError(t, err)
				require.NoDirExists(t, volume.path(job.DataPath))
				require.NoDirExists(t, volume.path(job.OutputPath))
			}
			pending, err := repo.HasFileCleanupJob(ctx, job.TaskID)
			require.NoError(t, err)
			require.False(t, pending)
		})
	}
}

func TestCleanupCheckDefersUncommittedCreationIntegration(t *testing.T) {
	repo, db := newTaskIntegrationRepository(t)
	ctx, task := integrationTask(t, repo, domain.TaskStatusCreated)
	job, err := domain.NewFileCleanupJob(task.ID, task.DataPath, task.OutputPath)
	require.NoError(t, err)
	for _, commit := range []bool{false, true} {
		t.Run(map[bool]string{false: "rollback", true: "commit"}[commit], func(t *testing.T) {
			require.NoError(t, repo.QueueFileCleanupJob(ctx, job))
			tx, err := db.Begin(ctx)
			require.NoError(t, err)
			defer tx.Rollback(ctx)
			_, err = tx.Exec(ctx, "DELETE FROM file_cleanup WHERE task_id=$1", task.ID)
			require.NoError(t, err)
			// Do not read the old MVCC version, and do not wait for the creator.
			_, err = repo.HasFileCleanupJob(ctx, task.ID)
			var lockError *pgconn.PgError
			require.ErrorAs(t, err, &lockError)
			require.Equal(t, "55P03", lockError.Code, "NOWAIT must defer cleanup")
			if commit {
				require.NoError(t, tx.Commit(ctx))
			} else {
				require.NoError(t, tx.Rollback(ctx))
			}
			pending, err := repo.HasFileCleanupJob(ctx, task.ID)
			require.NoError(t, err)
			require.Equal(t, !commit, pending)
			if !commit {
				// The standalone read has returned: no transaction or row lock remains.
				tx, err := db.Begin(ctx)
				require.NoError(t, err)
				defer tx.Rollback(ctx)
				_, err = tx.Exec(ctx, "SELECT task_id FROM file_cleanup WHERE task_id=$1 FOR UPDATE NOWAIT", task.ID)
				require.NoError(t, err)
				require.NoError(t, tx.Rollback(ctx))
				require.NoError(t, repo.DeleteFileCleanupJob(ctx, task.ID))
			}
		})
	}
}

func TestCreationCleanupAfterProcessDeathIntegration(t *testing.T) {
	if dsn := os.Getenv("PINN_CREATION_CHILD_DSN"); dsn != "" {
		cfg, err := pgx.ParseConfig(dsn)
		require.NoError(t, err)
		cfg.Database = os.Getenv("PINN_CREATION_CHILD_DB")
		repo, err := New(Config{
			User:     cfg.User,
			Password: cfg.Password,
			Host:     cfg.Host,
			Port:     int(cfg.Port),
			DB:       cfg.Database,
			SslMode:  "disable",
			Timeout:  10 * time.Second,
		}, zerolog.Nop())
		require.NoError(t, err)
		defer repo.Close()
		volume := creationVolume{root: os.Getenv("PINN_CREATION_CHILD_VOLUME")}
		files := observingCreationVolume{
			creationVolume: volume,
			hook: func() error {
				_, err := os.Stdout.Write([]byte{1})
				if err != nil {
					return err
				}
				<-time.After(time.Hour) // Parent kills this process while it holds the filesystem lock.
				return nil
			},
		}
		ctx := corefixtures.NewAuthInfo().WithContext(t.Context())
		_, err = tasksCreate.New(repo, files).CreateTask(ctx, tasksCreate.Input{
			Name: "interrupted upload",
			Mode: domain.TaskModeTrain,
		})
		require.NoError(t, err)
		return
	}
	repo, db := newTaskIntegrationRepository(t)
	ctx, cancel := context.WithTimeout(t.Context(), 20*time.Second)
	defer cancel()
	volume := creationVolume{root: t.TempDir()}
	child := exec.CommandContext(ctx, os.Args[0], "-test.run=^TestCreationCleanupAfterProcessDeathIntegration$")
	child.Env = append(os.Environ(),
		"PINN_CREATION_CHILD_DSN="+db.Config().ConnString(),
		"PINN_CREATION_CHILD_DB="+db.Config().Database,
		"PINN_CREATION_CHILD_VOLUME="+volume.root,
	)
	ready, err := child.StdoutPipe()
	require.NoError(t, err)
	require.NoError(t, child.Start())
	defer func() {
		if child.ProcessState == nil {
			_ = child.Process.Kill()
			_ = child.Wait()
		}
	}()
	signal := make([]byte, 1)
	_, err = io.ReadFull(ready, signal)
	require.NoError(t, err, "child must reach staged filesystem work")
	require.Equal(t, byte(1), signal[0], "child must signal readiness, not print a test failure")
	jobs, err := repo.GetFileCleanupJobs(ctx, uuid.Nil, limit.MustNew(32))
	require.NoError(t, err)
	require.Len(t, jobs, 1)
	require.DirExists(t, volume.path(jobs[0].DataPath))
	_, err = tasksCleanup.New(repo, volume, limit.MustNew(32)).CleanupPending(ctx, tasksCleanup.Input{})
	require.ErrorIs(t, err, errs.ErrConflict)
	require.NoError(t, child.Process.Kill())
	require.Error(t, child.Wait())
	// No graceful shutdown or deferred cleanup occurred in the creator process.
	_, err = tasksCleanup.New(repo, volume, limit.MustNew(32)).CleanupPending(ctx, tasksCleanup.Input{})
	require.NoError(t, err)
	require.NoDirExists(t, volume.path(jobs[0].DataPath))
	require.NoDirExists(t, volume.path(jobs[0].OutputPath))
	pending, err := repo.HasFileCleanupJob(ctx, jobs[0].TaskID)
	require.NoError(t, err)
	require.False(t, pending)
}
