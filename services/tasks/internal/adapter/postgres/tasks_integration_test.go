package postgres

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/rs/zerolog"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/AridanWarlock/pinnAutomizer/pkg/core"
	"github.com/AridanWarlock/pinnAutomizer/pkg/core/corefixtures"
	"github.com/AridanWarlock/pinnAutomizer/pkg/errs"
	"github.com/AridanWarlock/pinnAutomizer/pkg/test"
	"github.com/AridanWarlock/pinnAutomizer/tasks/internal/adapter/fileservice"
	"github.com/AridanWarlock/pinnAutomizer/tasks/internal/domain"
	tasksAfterRun "github.com/AridanWarlock/pinnAutomizer/tasks/internal/usecases/v1/tasks/afterRun"
	tasksDelete "github.com/AridanWarlock/pinnAutomizer/tasks/internal/usecases/v1/tasks/delete"
	tasksOnRun "github.com/AridanWarlock/pinnAutomizer/tasks/internal/usecases/v1/tasks/onRun"
	tasksRun "github.com/AridanWarlock/pinnAutomizer/tasks/internal/usecases/v1/tasks/run"
)

// The opt-in DSN must point to a disposable test cluster with CREATE DATABASE permission.
// Every fixture creates and drops its own database; existing databases are never migrated.
func newTaskIntegrationRepository(t *testing.T) (*Repository, *pgx.Conn) {
	t.Helper()
	dsn := os.Getenv("PINN_TEST_POSTGRES_DSN")
	if dsn == "" {
		t.Skip("set PINN_TEST_POSTGRES_DSN to a disposable PostgreSQL cluster")
	}
	ctx := context.Background()
	cfg, err := pgx.ParseConfig(dsn)
	require.NoError(t, err)
	admin, err := pgx.ConnectConfig(ctx, cfg)
	require.NoError(t, err)
	name := "pinn_test_" + uuid.New().String()
	quoted := pgx.Identifier{name}.Sanitize()
	_, err = admin.Exec(ctx, "CREATE DATABASE "+quoted)
	require.NoError(t, err)
	t.Cleanup(func() {
		cleanupCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		_, err := admin.Exec(cleanupCtx, "DROP DATABASE "+quoted)
		assert.NoError(t, err)
		assert.NoError(t, admin.Close(cleanupCtx))
	})
	testCfg := cfg.Copy()
	testCfg.Database = name
	db, err := pgx.ConnectConfig(ctx, testCfg)
	require.NoError(t, err)
	t.Cleanup(func() { assert.NoError(t, db.Close(context.Background())) })
	_, err = db.Exec(ctx, `
 CREATE TYPE task_status_enum AS ENUM ('created','in_queue','running','done','error');
 CREATE TABLE tasks (
 id uuid PRIMARY KEY, name text NOT NULL, description text,
 mode text NOT NULL, status task_status_enum NOT NULL, error text,
 data_path text NOT NULL, output_path text NOT NULL, plot_path text,
 user_id uuid NOT NULL, created_at timestamptz NOT NULL
 );
 CREATE TABLE events (
 id_key text PRIMARY KEY, topic text NOT NULL, data bytea NOT NULL, created_at timestamptz NOT NULL
 );`)
	require.NoError(t, err)
	migration, err := os.ReadFile("../../../migrations/20261013190000_create_task_run_requests.sql")
	require.NoError(t, err)
	up := strings.Split(string(migration), "-- +goose Down")[0]
	_, err = db.Exec(ctx, up)
	require.NoError(t, err)

	repo, err := New(Config{User: cfg.User, Password: cfg.Password, Host: cfg.Host, Port: int(cfg.Port), DB: name, SslMode: "disable", Timeout: 10 * time.Second}, zerolog.Nop())
	require.NoError(t, err)
	t.Cleanup(repo.Close)
	return repo, db
}

func integrationTask(t *testing.T, repo *Repository, status domain.TaskStatus) (context.Context, domain.Task) {
	t.Helper()
	auth := corefixtures.NewAuthInfo()
	ctx := auth.WithContext(test.ContextWithZeroLogger())
	ctx = core.IdempotencyKey(uuid.NewString()).WithContext(ctx)
	task, err := domain.NewTask("task", auth.UserID, domain.TaskModeTrain, nil)
	require.NoError(t, err)
	task.Status = status
	dir := t.TempDir()
	task.DataPath = filepath.Join(dir, "data")
	task.OutputPath = filepath.Join(dir, "output")
	require.NoError(t, os.MkdirAll(task.DataPath, 0700))
	require.NoError(t, os.MkdirAll(task.OutputPath, 0700))
	task, err = repo.CreateTask(ctx, task)
	require.NoError(t, err)
	return ctx, task
}

// Hold the real row lock until the test observes a competing PostgreSQL transaction waiting.
type heldTaskRepository struct {
	*Repository
	locked  chan struct{}
	release chan struct{}
}

func (r *heldTaskRepository) GetTaskByIDAndUserIDForUpdate(ctx context.Context, id, userID uuid.UUID) (domain.Task, error) {
	task, err := r.Repository.GetTaskByIDAndUserIDForUpdate(ctx, id, userID)
	if err != nil {
		return task, err
	}
	close(r.locked)
	select {
	case <-r.release:
		return task, nil
	case <-ctx.Done():
		return domain.Task{}, ctx.Err()
	}
}
func (r *heldTaskRepository) GetTaskByIDForUpdate(ctx context.Context, id uuid.UUID) (domain.Task, error) {
	task, err := r.Repository.GetTaskByIDForUpdate(ctx, id)
	if err != nil {
		return task, err
	}
	close(r.locked)
	select {
	case <-r.release:
		return task, nil
	case <-ctx.Done():
		return domain.Task{}, ctx.Err()
	}
}
func waitTaskLock(t *testing.T, db *pgx.Conn) {
	t.Helper()
	require.Eventually(t, func() bool {
		var waiting bool
		err := db.QueryRow(context.Background(), `SELECT EXISTS (
   SELECT 1 FROM pg_stat_activity WHERE datname=current_database() AND wait_event_type='Lock'
  )`).Scan(&waiting)
		return err == nil && waiting
	}, 5*time.Second, 10*time.Millisecond, "competing transaction must wait for the task row lock")
}
func awaitTaskLock(t *testing.T, held *heldTaskRepository) {
	t.Helper()
	select {
	case <-held.locked:
	case <-time.After(5 * time.Second):
		t.Fatal("first transaction did not acquire task lock")
	}
}

func TestTaskIntegrationConcurrentRunDelete(t *testing.T) {
	for _, runFirst := range []bool{true, false} {
		name := "delete-first"
		if runFirst {
			name = "run-first"
		}
		t.Run(name, func(t *testing.T) {
			repo, db := newTaskIntegrationRepository(t)
			ctx, task := integrationTask(t, repo, domain.TaskStatusCreated)
			held := &heldTaskRepository{Repository: repo, locked: make(chan struct{}), release: make(chan struct{})}
			var once sync.Once
			release := func() { once.Do(func() { close(held.release) }) }
			defer release()
			first := make(chan error, 1)
			second := make(chan error, 1)
			files := fileservice.NewFileService()
			if runFirst {
				go func() {
					first <- tasksRun.New(held).RunTask(ctx, tasksRun.Input{TaskID: task.ID})
				}()
			} else {
				go func() { first <- tasksDelete.New(held, files).DeleteTask(ctx, tasksDelete.Input{TaskID: task.ID}) }()
			}
			awaitTaskLock(t, held)
			if runFirst {
				go func() { second <- tasksDelete.New(repo, files).DeleteTask(ctx, tasksDelete.Input{TaskID: task.ID}) }()
			} else {
				go func() {
					second <- tasksRun.New(repo).RunTask(ctx, tasksRun.Input{TaskID: task.ID})
				}()
			}
			waitTaskLock(t, db)
			release()
			require.NoError(t, <-first)
			if runFirst {
				require.ErrorIs(t, <-second, errs.ErrConflict)
				stored, err := repo.GetTaskByID(ctx, task.ID)
				require.NoError(t, err)
				assert.Equal(t, domain.TaskStatusInQueue, stored.Status)
				assert.DirExists(t, task.DataPath)
				assert.DirExists(t, task.OutputPath)
			} else {
				require.ErrorIs(t, <-second, errs.ErrNotFound)
				_, err := repo.GetTaskByID(ctx, task.ID)
				require.ErrorIs(t, err, errs.ErrNotFound)
				assert.NoDirExists(t, task.DataPath)
				assert.NoDirExists(t, task.OutputPath)
			}
			var count int
			require.NoError(t, db.QueryRow(ctx, "SELECT COUNT(*) FROM events").Scan(&count))
			if runFirst {
				assert.Equal(t, 1, count)
			} else {
				assert.Zero(t, count)
			}
		})
	}
}

func TestTaskIntegrationRepeatedAndReorderedEvents(t *testing.T) {
	repo, _ := newTaskIntegrationRepository(t)
	files := fileservice.NewFileService()
	onRun := tasksOnRun.New(repo)
	afterRun := tasksAfterRun.New(repo, files)
	for _, finishFirst := range []bool{false, true} {
		name := "start-first"
		if finishFirst {
			name = "finish-first"
		}
		t.Run(name, func(t *testing.T) {
			ctx, task := integrationTask(t, repo, domain.TaskStatusInQueue)
			plot := filepath.Join(task.OutputPath, "fig.pdf")
			require.NoError(t, os.WriteFile(plot, []byte("plot"), 0600))
			if !finishFirst {
				require.NoError(t, onRun.UpdateTaskOnRun(ctx, tasksOnRun.Input{ID: task.ID}))
				require.NoError(t, onRun.UpdateTaskOnRun(ctx, tasksOnRun.Input{ID: task.ID}))
			}
			require.NoError(t, afterRun.UpdateTaskAfterRun(ctx, tasksAfterRun.Input{ID: task.ID}))
			// A late conflicting completion must not replace the original result or plot.
			require.NoError(t, os.Remove(plot))
			failure := "late error"
			require.NoError(t, afterRun.UpdateTaskAfterRun(ctx, tasksAfterRun.Input{ID: task.ID, Error: &failure}))
			require.NoError(t, onRun.UpdateTaskOnRun(ctx, tasksOnRun.Input{ID: task.ID}))
			stored, err := repo.GetTaskByID(ctx, task.ID)
			require.NoError(t, err)
			assert.Equal(t, domain.TaskStatusDone, stored.Status)
			assert.Nil(t, stored.Error)
			require.NotNil(t, stored.PlotPath)
			assert.Equal(t, plot, *stored.PlotPath)
		})
	}
}

func TestTaskIntegrationConcurrentCompletionPreservesFirstResult(t *testing.T) {
	repo, db := newTaskIntegrationRepository(t)
	ctx, task := integrationTask(t, repo, domain.TaskStatusRunning)
	held := &heldTaskRepository{Repository: repo, locked: make(chan struct{}), release: make(chan struct{})}
	var once sync.Once
	release := func() { once.Do(func() { close(held.release) }) }
	defer release()
	failure := "first error"
	first := make(chan error, 1)
	second := make(chan error, 1)
	files := fileservice.NewFileService()
	go func() {
		first <- tasksAfterRun.New(held, files).UpdateTaskAfterRun(ctx, tasksAfterRun.Input{ID: task.ID, Error: &failure})
	}()
	awaitTaskLock(t, held)
	go func() {
		second <- tasksAfterRun.New(repo, files).UpdateTaskAfterRun(ctx, tasksAfterRun.Input{ID: task.ID})
	}()
	waitTaskLock(t, db)
	release()
	require.NoError(t, <-first)
	require.NoError(t, <-second)
	stored, err := repo.GetTaskByID(ctx, task.ID)
	require.NoError(t, err)
	assert.Equal(t, domain.TaskStatusError, stored.Status)
	assert.Equal(t, &failure, stored.Error)
}
