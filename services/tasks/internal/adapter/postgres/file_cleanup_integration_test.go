package postgres

import (
	"context"
	"errors"
	"os"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/rs/zerolog"
	"github.com/stretchr/testify/require"

	"github.com/AridanWarlock/pinnAutomizer/pkg/core/limit"
	"github.com/AridanWarlock/pinnAutomizer/pkg/errs"
	"github.com/AridanWarlock/pinnAutomizer/tasks/internal/domain"
	tasksCleanup "github.com/AridanWarlock/pinnAutomizer/tasks/internal/usecases/v1/tasks/cleanup"
	tasksDelete "github.com/AridanWarlock/pinnAutomizer/tasks/internal/usecases/v1/tasks/delete"
)

type cleanupFileService struct {
	remove func(string) error
}

func (f cleanupFileService) RemoveAll(path string) error {
	return f.remove(path)
}

func TestFileCleanupAtomicDeletePartialFailureAndReopenIntegration(t *testing.T) {
	repo, db := newTaskIntegrationRepository(t)
	ctx, task := integrationTask(t, repo, domain.TaskStatusCreated)
	_, err := db.Exec(ctx, `
 CREATE FUNCTION reject_delete() RETURNS trigger LANGUAGE plpgsql AS $$
 BEGIN RAISE EXCEPTION 'injected delete failure'; END $$;
 CREATE TRIGGER reject_delete BEFORE DELETE ON tasks FOR EACH ROW EXECUTE FUNCTION reject_delete();
 `)
	require.NoError(t, err)
	err = tasksDelete.New(repo).DeleteTask(ctx, tasksDelete.Input{TaskID: task.ID})
	require.ErrorContains(t, err, "injected delete failure")
	_, err = repo.GetTaskByID(ctx, task.ID)
	require.NoError(t, err)
	jobs, err := repo.GetFileCleanups(ctx, uuid.Nil, limit.MustNew(10))
	require.NoError(t, err)
	require.Empty(t, jobs, "failed delete must roll back the queued cleanup")
	require.DirExists(t, task.DataPath)
	require.DirExists(t, task.OutputPath)
	_, err = db.Exec(ctx, "DROP TRIGGER reject_delete ON tasks")
	require.NoError(t, err)

	require.NoError(t, tasksDelete.New(repo).DeleteTask(ctx, tasksDelete.Input{TaskID: task.ID}))
	_, err = repo.GetTaskByID(ctx, task.ID)
	require.ErrorIs(t, err, errs.ErrNotFound)
	require.DirExists(t, task.DataPath, "HTTP deletion commits before filesystem work")
	require.DirExists(t, task.OutputPath)
	jobs, err = repo.GetFileCleanups(ctx, uuid.Nil, limit.MustNew(10))
	require.NoError(t, err)
	require.Len(t, jobs, 1)
	require.NoError(t, tasksDelete.New(repo).DeleteTask(ctx, tasksDelete.Input{TaskID: task.ID}), "replay after a lost HTTP response must succeed")
	require.Equal(t, task.ID, jobs[0].TaskID)
	files := cleanupFileService{
		remove: func(path string) error {
			var openTransactions int
			err := db.QueryRow(ctx, "SELECT COUNT(*) FROM pg_stat_activity WHERE datname=current_database() AND state='idle in transaction'").Scan(&openTransactions)
			require.NoError(t, err)
			require.Zero(t, openTransactions, "filesystem work must not keep a PostgreSQL transaction open")
			if path == task.OutputPath {
				return errors.New("output filesystem unavailable")
			}
			return os.RemoveAll(path)
		},
	}
	_, err = tasksCleanup.New(repo, files, limit.MustNew(10)).CleanupPending(ctx, tasksCleanup.Input{})
	require.ErrorContains(t, err, "output filesystem unavailable")
	require.NoDirExists(t, task.DataPath)
	require.DirExists(t, task.OutputPath)
	jobs, err = repo.GetFileCleanups(ctx, uuid.Nil, limit.MustNew(10))
	require.NoError(t, err)
	require.Len(t, jobs, 1)

	_, err = db.Exec(ctx, `
 CREATE FUNCTION reject_cleanup_ack() RETURNS trigger LANGUAGE plpgsql AS $$
 BEGIN RAISE EXCEPTION 'injected ack failure'; END $$;
 CREATE TRIGGER reject_cleanup_ack BEFORE DELETE ON file_cleanup FOR EACH ROW EXECUTE FUNCTION reject_cleanup_ack();
 `)
	require.NoError(t, err)
	files.remove = os.RemoveAll
	_, err = tasksCleanup.New(repo, files, limit.MustNew(10)).CleanupPending(ctx, tasksCleanup.Input{})
	require.ErrorContains(t, err, "injected ack failure")
	require.NoDirExists(t, task.DataPath)
	require.NoDirExists(t, task.OutputPath)
	repo.Close()
	cfg := db.Config()
	reopened, err := New(Config{
		User:     cfg.User,
		Password: cfg.Password,
		Host:     cfg.Host,
		Port:     int(cfg.Port),
		DB:       cfg.Database,
		SslMode:  "disable",
		Timeout:  10 * time.Second,
	}, zerolog.Nop())
	require.NoError(t, err)
	defer reopened.Close()
	jobs, err = reopened.GetFileCleanups(ctx, uuid.Nil, limit.MustNew(10))
	require.NoError(t, err)
	require.Len(t, jobs, 1, "an ack failure after physical deletion must remain retryable")
	_, err = db.Exec(ctx, "DROP TRIGGER reject_cleanup_ack ON file_cleanup")
	require.NoError(t, err)
	uc := tasksCleanup.New(reopened, files, limit.MustNew(10))
	_, err = uc.CleanupPending(ctx, tasksCleanup.Input{})
	require.NoError(t, err, "already absent directories are a successful retry")
	out, err := uc.CleanupPending(ctx, tasksCleanup.Input{})
	require.NoError(t, err)
	require.Equal(t, uuid.Nil, out.NextTaskID)
	jobs, err = reopened.GetFileCleanups(ctx, uuid.Nil, limit.MustNew(10))
	require.NoError(t, err)
	require.Empty(t, jobs)
	require.NoError(t, tasksDelete.New(reopened).DeleteTask(ctx, tasksDelete.Input{TaskID: task.ID}))
	jobs, err = reopened.GetFileCleanups(ctx, uuid.Nil, limit.MustNew(10))
	require.NoError(t, err)
	require.Empty(t, jobs, "replay after cleanup must not recreate a job")
}

func TestFileCleanupInvalidRowIsPersistenceFailure(t *testing.T) {
	t.Parallel()
	require.Panics(t, func() {
		_ = (fileCleanupRow{TaskID: uuid.New(), DataPath: "/", OutputPath: "/"}).toDomain()
	})
}

func TestConcurrentFileCleanupIsIdempotentIntegration(t *testing.T) {
	repo, _ := newTaskIntegrationRepository(t)
	authCtx, task := integrationTask(t, repo, domain.TaskStatusCreated)
	ctx, cancel := context.WithTimeout(authCtx, 10*time.Second)
	defer cancel()
	require.NoError(t, tasksDelete.New(repo).DeleteTask(ctx, tasksDelete.Input{TaskID: task.ID}))
	ready := make(chan struct{}, 2)
	release := make(chan struct{})
	var once sync.Once
	unlock := func() { once.Do(func() { close(release) }) }
	var workers sync.WaitGroup
	defer workers.Wait()
	defer unlock()
	files := cleanupFileService{
		remove: func(path string) error {
			if path == task.DataPath {
				ready <- struct{}{}
				select {
				case <-release:
				case <-ctx.Done():
					return ctx.Err()
				}
			}
			return os.RemoveAll(path)
		},
	}
	uc := tasksCleanup.New(repo, files, limit.MustNew(10))
	results := make(chan error, 2)
	for range 2 {
		workers.Go(func() {
			_, err := uc.CleanupPending(ctx, tasksCleanup.Input{})
			results <- err
		})
	}
	for range 2 {
		select {
		case <-ready:
		case err := <-results:
			t.Fatalf("cleanup returned before both workers read the job: %v", err)
		case <-ctx.Done():
			t.Fatal("workers did not reach the cleanup barrier")
		}
	}
	unlock()
	for range 2 {
		require.NoError(t, <-results)
	}
	require.NoDirExists(t, task.DataPath)
	require.NoDirExists(t, task.OutputPath)
	jobs, err := repo.GetFileCleanups(ctx, uuid.Nil, limit.MustNew(10))
	require.NoError(t, err)
	require.Empty(t, jobs)
}
