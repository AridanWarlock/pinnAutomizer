package tasksObserve

import (
	"context"
	"database/sql"
	"os"
	"path/filepath"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/pressly/goose/v3"
	"github.com/stretchr/testify/require"

	"github.com/AridanWarlock/pinnAutomizer/pkg/core"
	"github.com/AridanWarlock/pinnAutomizer/pkg/core/limit"
	"github.com/AridanWarlock/pinnAutomizer/solver/internal/adapter/sqlite"
	"github.com/AridanWarlock/pinnAutomizer/solver/internal/domain"
	"github.com/AridanWarlock/pinnAutomizer/solver/internal/domain/fixtures"
)

func journal(t *testing.T) (sqlite.Config, *sql.DB) {
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
	t.Cleanup(func() { require.NoError(t, db.Close()) })
	provider, err := goose.NewProvider(goose.DialectSQLite3, db, os.DirFS(dir), goose.WithDisableGlobalRegistry(true))
	require.NoError(t, err)
	_, err = provider.Up(t.Context())
	require.NoError(t, err)
	return cfg, db
}

func registerStarted(t *testing.T, repo *sqlite.Repository) domain.Attempt {
	t.Helper()
	a := fixtures.NewAttempt(t)
	_, _, err := repo.RegisterAttempt(t.Context(), a)
	require.NoError(t, err)
	_, err = repo.RecordContainer(t.Context(), a.RunID, "container-id")
	require.NoError(t, err)
	_, err = repo.RequestStart(t.Context(), a.RunID, time.Now(), time.Minute)
	require.NoError(t, err)
	a, err = repo.GetAttempt(t.Context(), a.RunID)
	require.NoError(t, err)
	return a
}

func TestObserveTaskRollbackReopenAndReplay(t *testing.T) {
	cfg, db := journal(t)
	repo, err := sqlite.New(cfg)
	require.NoError(t, err)
	defer repo.Close()
	a := registerStarted(t, repo)
	runner := &MockPinnRunner{
		ObserveAttemptFunc: func(_ context.Context, a domain.Attempt) (domain.Attempt, error) { return completed(t, a), nil },
	}
	_, err = db.Exec("CREATE TRIGGER reject_event BEFORE INSERT ON outbox BEGIN SELECT RAISE(ABORT,'injected failure'); END;")
	require.NoError(t, err)
	_, err = New(repo, runner).ObserveTask(t.Context(), Input{RunID: a.RunID})
	require.Error(t, err)
	stored, err := repo.GetAttempt(t.Context(), a.RunID)
	require.NoError(t, err)
	require.True(t, stored.CanObserve(), "outbox failure must roll back completion")
	events, err := repo.GetPendingEvents(t.Context(), limit.MustNew(10))
	require.NoError(t, err)
	require.Empty(t, events)
	_, err = db.Exec("DROP TRIGGER reject_event")
	require.NoError(t, err)
	out, err := New(repo, runner).ObserveTask(t.Context(), Input{RunID: a.RunID})
	require.NoError(t, err)
	require.Equal(t, core.Changed(true), out.Finished)
	require.NoError(t, repo.Close())
	repo, err = sqlite.New(cfg)
	require.NoError(t, err)
	defer repo.Close()
	events, err = repo.GetPendingEvents(t.Context(), limit.MustNew(10))
	require.NoError(t, err)
	require.Len(t, events, 1)
	expected, err := newCompletionEvent(completed(t, a))
	require.NoError(t, err)
	require.Equal(t, expected, events[0])
	// Even after outbox delivery and deletion, a replay cannot recreate the event.
	require.NoError(t, repo.DeleteEvents(t.Context(), []uuid.UUID{events[0].ID}))
	callsBefore := len(runner.ObserveAttemptCalls())
	out, err = New(repo, runner).ObserveTask(t.Context(), Input{RunID: a.RunID})
	require.NoError(t, err)
	require.Equal(t, core.Changed(false), out.Finished)
	require.Len(t, runner.ObserveAttemptCalls(), callsBefore)
	events, err = repo.GetPendingEvents(t.Context(), limit.MustNew(10))
	require.NoError(t, err)
	require.Empty(t, events)
}

func TestObserveTaskConcurrentCompletionCreatesOneEvent(t *testing.T) {
	cfg, _ := journal(t)
	repo, err := sqlite.New(cfg)
	require.NoError(t, err)
	defer repo.Close()
	a := registerStarted(t, repo)
	ctx, cancel := context.WithTimeout(t.Context(), 5*time.Second)
	defer cancel()
	const observers = 8
	entered := make(chan struct{}, observers)
	release := make(chan struct{})
	runner := &MockPinnRunner{
		ObserveAttemptFunc: func(ctx context.Context, a domain.Attempt) (domain.Attempt, error) {
			entered <- struct{}{}
			select {
			case <-release:
				code := 0
				_, err := a.Finish(&code, nil, time.Now())
				return a, err
			case <-ctx.Done():
				return domain.Attempt{}, ctx.Err()
			}
		},
	}
	uc := New(repo, runner)
	type result struct {
		out Output
		err error
	}
	results := make(chan result, observers)
	var workers sync.WaitGroup
	for range observers {
		workers.Go(func() {
			out, err := uc.ObserveTask(ctx, Input{RunID: a.RunID})
			results <- result{out: out, err: err}
		})
	}
	// All callers read the unfinished attempt before any one may commit its result.
	for range observers {
		select {
		case <-entered:
		case <-ctx.Done():
			t.Fatal(ctx.Err())
		}
	}
	close(release)
	workers.Wait()
	close(results)
	changed := 0
	for res := range results {
		require.NoError(t, res.err)
		if res.out.Finished {
			changed++
		}
	}
	require.Equal(t, 1, changed)
	events, err := repo.GetPendingEvents(t.Context(), limit.MustNew(10))
	require.NoError(t, err)
	require.Len(t, events, 1)
}
