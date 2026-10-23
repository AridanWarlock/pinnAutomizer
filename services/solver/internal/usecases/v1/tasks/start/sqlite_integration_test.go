package tasksStart

import (
	"context"
	"database/sql"
	"errors"
	"os"
	"path/filepath"
	"sync"
	"testing"
	"time"

	"github.com/pressly/goose/v3"
	"github.com/stretchr/testify/require"

	"github.com/AridanWarlock/pinnAutomizer/pkg/core/limit"
	"github.com/AridanWarlock/pinnAutomizer/pkg/errs"
	"github.com/AridanWarlock/pinnAutomizer/solver/internal/adapter/sqlite"
	"github.com/AridanWarlock/pinnAutomizer/solver/internal/domain"
	"github.com/AridanWarlock/pinnAutomizer/solver/internal/domain/fixtures"
)

func TestStartTaskConcurrentCallersAndReplayOnDisk(t *testing.T) {
	dir, err := filepath.Abs("../../../../../migrations")
	require.NoError(t, err)
	path := filepath.Join(t.TempDir(), "journal.db")
	db, err := sql.Open("sqlite", path)
	require.NoError(t, err)
	defer db.Close()
	provider, err := goose.NewProvider(goose.DialectSQLite3, db, os.DirFS(dir), goose.WithDisableGlobalRegistry(true))
	require.NoError(t, err)
	_, err = provider.Up(t.Context())
	require.NoError(t, err)
	cfg := sqlite.Config{Path: path, MigrationsDir: dir, BusyTimeout: time.Second}
	repo, err := sqlite.New(cfg)
	require.NoError(t, err)
	defer repo.Close()
	a := fixtures.NewAttempt(t)
	_, _, err = repo.RegisterAttempt(t.Context(), a)
	require.NoError(t, err)
	runner := &MockPinnRunner{
		EnsureContainerFunc: func(context.Context, domain.Attempt) (string, error) { return "container", nil },
		StartContainerFunc:  func(context.Context, domain.Attempt) error { return domain.ErrStartOutcomeUnknown },
	}
	uc := New(repo, runner, time.Minute, limit.MustNew(1))
	gate := make(chan struct{})
	var workers sync.WaitGroup
	results := make(chan error, 8)
	for range cap(results) {
		workers.Go(func() {
			<-gate
			_, err := uc.StartTask(t.Context(), Input{RunID: a.RunID})
			results <- err
		})
	}
	close(gate)
	workers.Wait()
	close(results)
	for err := range results {
		if err != nil {
			// A competing SQLite transition may lose the conditional write/upgrade.
			require.True(t, errors.Is(err, errs.ErrConflict) || errors.Is(err, domain.ErrStartOutcomeUnknown), "%v", err)
		}
	}
	require.Len(t, runner.StartContainerCalls(), 1)
	require.NoError(t, repo.Close())
	repo, err = sqlite.New(cfg)
	require.NoError(t, err)
	defer repo.Close()
	_, err = New(repo, runner, time.Minute, limit.MustNew(1)).StartTask(t.Context(), Input{RunID: a.RunID})
	require.NoError(t, err)
	require.Len(t, runner.StartContainerCalls(), 1, "reopen must preserve the consumed intent")
}
