package sqlite

import (
	"bufio"
	"context"
	"database/sql"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/AridanWarlock/pinnAutomizer/pkg/core"
	"github.com/AridanWarlock/pinnAutomizer/pkg/core/limit"
	"github.com/AridanWarlock/pinnAutomizer/pkg/errs"
	shared "github.com/AridanWarlock/pinnAutomizer/pkg/sqlite"
	"github.com/AridanWarlock/pinnAutomizer/solver/internal/domain"
	"github.com/AridanWarlock/pinnAutomizer/solver/internal/domain/fixtures"
)

func sqliteClient(repo *Repository) *shared.SQLite {
	return repo.sqlite.(*shared.SQLite)
}

func config(t *testing.T) Config {
	t.Helper()
	dir, err := filepath.Abs("../../../migrations")
	require.NoError(t, err)
	cfg := Config{Path: filepath.Join(t.TempDir(), "journal.db"), BusyTimeout: 50 * time.Millisecond, MigrationsDir: dir}
	migrate(t, cfg.Path, os.DirFS(dir))
	return cfg
}

func open(t *testing.T, c Config) *Repository {
	t.Helper()
	r, err := New(c)
	require.NoError(t, err)
	t.Cleanup(func() {
		require.NoError(t, r.Close())
	})
	return r
}

func attempt(t *testing.T) domain.Attempt {
	t.Helper()
	a, err := domain.NewAttempt(uuid.New(), uuid.New(), "train", time.Now())
	require.NoError(t, err)
	return a
}

func event(t *testing.T, a domain.Attempt) domain.Event {
	t.Helper()
	id := uuid.New()
	message, err := core.NewProduceKafkaMessage("after-run", []byte(a.TaskID.String()), []byte("result"), core.KafkaHeaders{core.KafkaIdempotencyHeader: id.String()})
	require.NoError(t, err)
	event, err := domain.NewEvent(id, a.RunID, message)
	require.NoError(t, err)
	return event
}

func TestOwnershipAndPragmas(t *testing.T) {
	c := config(t)
	r := open(t, c)
	_, err := New(c)
	require.ErrorIs(t, err, ErrAlreadyOwned)
	var mode string
	require.NoError(t, sqliteClient(r).Database().QueryRow("PRAGMA journal_mode").Scan(&mode))
	require.Equal(t, "wal", mode)
	for pragma, want := range map[string]int{"synchronous": 2, "foreign_keys": 1, "busy_timeout": 50} {
		var got int
		require.NoError(t, sqliteClient(r).Database().QueryRow("PRAGMA "+pragma).Scan(&got))
		require.Equal(t, want, got)
	}
	require.NoError(t, r.Close())
	open(t, c)
}

func TestConcurrentRegistrationAndConflicts(t *testing.T) {
	r := open(t, config(t))
	a := attempt(t)
	var created atomic.Int32
	var wg sync.WaitGroup
	for range 20 {
		wg.Go(func() {
			stored, newRecord, err := r.RegisterAttempt(context.Background(), a)
			assert.NoError(t, err)
			assert.True(t, stored.Matches(a))
			if newRecord {
				created.Add(1)
			}
		})
	}
	wg.Wait()
	require.EqualValues(t, 1, created.Load())
	changed, err := domain.NewAttempt(a.RunID, a.TaskID, "predict", a.CreatedAt)
	require.NoError(t, err)
	_, _, err = r.RegisterAttempt(context.Background(), changed)
	require.ErrorIs(t, err, errs.ErrConflict)
	changed, err = domain.NewAttempt(uuid.New(), a.TaskID, a.Mode, a.CreatedAt)
	require.NoError(t, err)
	_, _, err = r.RegisterAttempt(context.Background(), changed)
	require.ErrorIs(t, err, errs.ErrConflict)
}

func TestFinishRollbackReplayAndReopen(t *testing.T) {
	ctx := context.Background()
	c := config(t)
	r := open(t, c)
	a := attempt(t)
	_, _, err := r.RegisterAttempt(ctx, a)
	require.NoError(t, err)
	e := event(t, a)
	code := 0
	_, err = sqliteClient(r).Database().Exec("CREATE TRIGGER reject_event BEFORE INSERT ON outbox BEGIN SELECT RAISE(ABORT,'injected failure'); END;")
	require.NoError(t, err)
	_, err = r.FinishAttempt(ctx, a.RunID, &code, nil, time.Now(), e)
	require.Error(t, err)
	stored, err := r.GetAttempt(ctx, a.RunID)
	require.NoError(t, err)
	require.Equal(t, domain.AttemptAccepted, stored.State)
	_, err = sqliteClient(r).Database().Exec("DROP TRIGGER reject_event")
	require.NoError(t, err)
	changed, err := r.FinishAttempt(ctx, a.RunID, &code, nil, time.Now(), e)
	require.NoError(t, err)
	require.True(t, bool(changed))
	failure := "late error"
	changed, err = r.FinishAttempt(ctx, a.RunID, nil, &failure, time.Now(), event(t, a))
	require.NoError(t, err)
	require.False(t, bool(changed))
	require.NoError(t, r.Close())
	r = open(t, c)
	stored, err = r.GetAttempt(ctx, a.RunID)
	require.NoError(t, err)
	require.Nil(t, stored.Error)
	require.Equal(t, 0, *stored.ExitCode)
	events, err := r.GetPendingEvents(ctx, limit.MustNew(10))
	require.NoError(t, err)
	require.Equal(t, []domain.Event{e}, events)
}

func TestStartIntentIsNotRepeated(t *testing.T) {
	ctx := context.Background()
	r := open(t, config(t))
	a := attempt(t)
	_, _, err := r.RegisterAttempt(ctx, a)
	require.NoError(t, err)
	_, err = r.RequestStart(ctx, a.RunID, time.Now(), time.Hour, fixtures.NewEvent(t, a))
	require.ErrorIs(t, err, errs.ErrConflict)
	changed, err := r.RecordContainer(ctx, a.RunID, "container")
	require.NoError(t, err)
	require.True(t, bool(changed))
	changed, err = r.RequestStart(ctx, a.RunID, time.Now(), time.Hour, fixtures.NewEvent(t, a))
	require.NoError(t, err)
	require.True(t, bool(changed))
	first, err := r.GetAttempt(ctx, a.RunID)
	require.NoError(t, err)
	changed, err = r.RequestStart(ctx, a.RunID, time.Now(), 2*time.Hour, fixtures.NewEvent(t, a))
	require.NoError(t, err)
	require.False(t, bool(changed))
	second, err := r.GetAttempt(ctx, a.RunID)
	require.NoError(t, err)
	require.Equal(t, first.Deadline, second.Deadline)
}

func TestBusyAndCanceledRegistration(t *testing.T) {
	cfg := config(t)
	r := open(t, cfg)
	ctx := context.Background()
	conn, err := sqliteClient(r).Database().Conn(ctx)
	require.NoError(t, err)
	// A separate connection emulates another SQLite client holding the write lock.
	external, err := sql.Open("sqlite", cfg.Path)
	require.NoError(t, err)
	defer external.Close()
	_ = conn.Close()
	_, err = external.Exec("BEGIN IMMEDIATE")
	require.NoError(t, err)
	_, _, err = r.RegisterAttempt(ctx, attempt(t))
	require.Error(t, err)
	_, err = external.Exec("ROLLBACK")
	require.NoError(t, err)
	canceled, cancel := context.WithCancel(ctx)
	cancel()
	_, _, err = r.RegisterAttempt(canceled, attempt(t))
	require.True(t, errors.Is(err, context.Canceled))
	_, _, err = r.RegisterAttempt(ctx, attempt(t))
	require.NoError(t, err)
}

func TestOwnerProcessHelper(t *testing.T) {
	path := os.Getenv("PINN_TEST_OWNER_PATH")
	if path == "" {
		return
	}
	r, err := New(Config{Path: path, BusyTimeout: time.Second, MigrationsDir: os.Getenv("PINN_TEST_MIGRATIONS_DIR")})
	require.NoError(t, err)
	a := attempt(t)
	_, _, err = r.RegisterAttempt(context.Background(), a)
	require.NoError(t, err)
	fmt.Println(a.RunID.String())
	select {}
}

func TestKilledOwnerReleasesLockAndPreservesRecord(t *testing.T) {
	c := config(t)
	cmd := exec.Command(os.Args[0], "-test.run=^TestOwnerProcessHelper$")
	cmd.Env = append(os.Environ(), "PINN_TEST_OWNER_PATH="+c.Path, "PINN_TEST_MIGRATIONS_DIR="+c.MigrationsDir)
	stdout, err := cmd.StdoutPipe()
	require.NoError(t, err)
	require.NoError(t, cmd.Start())
	t.Cleanup(func() {
		_ = cmd.Process.Kill()
		_ = cmd.Wait()
	})
	ready := make(chan string, 1)
	go func() {
		scanner := bufio.NewScanner(stdout)
		if scanner.Scan() {
			ready <- scanner.Text()
		} else {
			ready <- ""
		}
	}()
	var id string
	select {
	case id = <-ready:
	case <-time.After(10 * time.Second):
		t.Fatal("journal owner did not become ready")
	}
	runID, err := uuid.Parse(id)
	require.NoError(t, err)
	_, err = New(c)
	require.ErrorIs(t, err, ErrAlreadyOwned)
	require.NoError(t, cmd.Process.Kill())
	require.Error(t, cmd.Wait())
	r := open(t, c)
	_, err = r.GetAttempt(context.Background(), runID)
	require.NoError(t, err)
}
