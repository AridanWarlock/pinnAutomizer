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
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/AridanWarlock/pinnAutomizer/pkg/core"
	"github.com/AridanWarlock/pinnAutomizer/pkg/errs"
	"github.com/AridanWarlock/pinnAutomizer/solver/internal/domain"
)

func config(t *testing.T) Config {
	t.Helper()
	return Config{Path: filepath.Join(t.TempDir(), "journal.db"), BusyTimeout: 50 * time.Millisecond}
}
func open(t *testing.T, c Config) *Repository {
	t.Helper()
	r, err := New(c)
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, r.Close()) })
	return r
}
func attempt(t *testing.T) domain.Attempt {
	t.Helper()
	a, err := domain.NewAttempt(uuid.New(), uuid.New(), "train", time.Now())
	require.NoError(t, err)
	return a
}
func event(a domain.Attempt) domain.Event {
	id := uuid.New()
	return domain.Event{ID: id, RunID: a.RunID, Message: core.KafkaMessage{Topic: "after-run", Key: []byte(a.TaskID.String()), Value: []byte("result"), Headers: core.KafkaHeaders{"X-Idempotency-Key": id.String()}}}
}

func TestOwnershipAndPragmas(t *testing.T) {
	c := config(t)
	r := open(t, c)
	_, err := New(c)
	require.ErrorIs(t, err, ErrAlreadyOwned)
	var mode string
	require.NoError(t, r.db.QueryRow("PRAGMA journal_mode").Scan(&mode))
	require.Equal(t, "wal", mode)
	for pragma, want := range map[string]int{"synchronous": 2, "foreign_keys": 1, "busy_timeout": 50} {
		var got int
		require.NoError(t, r.db.QueryRow("PRAGMA "+pragma).Scan(&got))
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
	changed := a
	changed.Mode = "predict"
	_, _, err := r.RegisterAttempt(context.Background(), changed)
	require.ErrorIs(t, err, errs.ErrConflict)
	changed = a
	changed.RunID = uuid.New()
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
	e := event(a)
	code := 0
	_, err = r.db.Exec("CREATE TRIGGER reject_event BEFORE INSERT ON outbox BEGIN SELECT RAISE(ABORT,'injected failure'); END;")
	require.NoError(t, err)
	_, err = r.FinishAttempt(ctx, a.RunID, &code, nil, time.Now(), e)
	require.Error(t, err)
	stored, err := r.GetAttempt(ctx, a.RunID)
	require.NoError(t, err)
	require.Equal(t, domain.AttemptRegistered, stored.State)
	_, err = r.db.Exec("DROP TRIGGER reject_event")
	require.NoError(t, err)
	changed, err := r.FinishAttempt(ctx, a.RunID, &code, nil, time.Now(), e)
	require.NoError(t, err)
	require.True(t, changed)
	failure := "late error"
	changed, err = r.FinishAttempt(ctx, a.RunID, nil, &failure, time.Now(), event(a))
	require.NoError(t, err)
	require.False(t, changed)
	require.NoError(t, r.Close())
	r = open(t, c)
	stored, err = r.GetAttempt(ctx, a.RunID)
	require.NoError(t, err)
	require.Nil(t, stored.Error)
	require.Equal(t, 0, *stored.ExitCode)
	events, err := r.GetPendingEvents(ctx, 10)
	require.NoError(t, err)
	require.Equal(t, []domain.Event{e}, events)
}
func TestStartIntentIsNotRepeated(t *testing.T) {
	ctx := context.Background()
	r := open(t, config(t))
	a := attempt(t)
	_, _, err := r.RegisterAttempt(ctx, a)
	require.NoError(t, err)
	_, err = r.RequestStart(ctx, a.RunID, time.Now(), time.Hour)
	require.ErrorIs(t, err, errs.ErrConflict)
	changed, err := r.RecordContainer(ctx, a.RunID, "container")
	require.NoError(t, err)
	require.True(t, changed)
	changed, err = r.RequestStart(ctx, a.RunID, time.Now(), time.Hour)
	require.NoError(t, err)
	require.True(t, changed)
	first, err := r.GetAttempt(ctx, a.RunID)
	require.NoError(t, err)
	changed, err = r.RequestStart(ctx, a.RunID, time.Now(), 2*time.Hour)
	require.NoError(t, err)
	require.False(t, changed)
	second, err := r.GetAttempt(ctx, a.RunID)
	require.NoError(t, err)
	require.Equal(t, first.Deadline, second.Deadline)
}
func TestBusyAndCanceledRegistration(t *testing.T) {
	r := open(t, config(t))
	ctx := context.Background()
	conn, err := r.db.Conn(ctx)
	require.NoError(t, err)
	// A separate connection emulates another SQLite client holding the write lock.
	external, err := sql.Open("sqlite", strings.TrimSuffix(r.owner.Name(), ".lock"))
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
	r, err := New(Config{Path: path, BusyTimeout: time.Second})
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
	cmd.Env = append(os.Environ(), "PINN_TEST_OWNER_PATH="+c.Path)
	stdout, err := cmd.StdoutPipe()
	require.NoError(t, err)
	require.NoError(t, cmd.Start())
	t.Cleanup(func() { _ = cmd.Process.Kill(); _ = cmd.Wait() })
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
