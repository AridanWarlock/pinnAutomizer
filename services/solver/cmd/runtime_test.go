package main

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

	"github.com/AridanWarlock/pinnAutomizer/pkg/kafka"
	"github.com/AridanWarlock/pinnAutomizer/solver/internal/adapter/mlrunner"
	"github.com/AridanWarlock/pinnAutomizer/solver/internal/adapter/sqlite"
	"github.com/AridanWarlock/pinnAutomizer/solver/internal/config"
)

func testConfig(t *testing.T) (config.Config, *sql.DB) {
	t.Helper()
	dir, err := filepath.Abs("../migrations")
	require.NoError(t, err)
	cfg := config.Config{
		SQLite: sqlite.Config{
			Path:          filepath.Join(t.TempDir(), "journal.db"),
			MigrationsDir: dir,
			BusyTimeout:   time.Second,
		},
		Recovery: config.Recovery{
			ActiveLimit:    1,
			BatchSize:      2,
			Interval:       10 * time.Millisecond,
			AttemptTimeout: 5 * time.Second,
		},
		OutboxInterval: 10 * time.Millisecond,
		PinnRunner: mlrunner.Config{
			ContainerNamePrefix: "pinn-runtime-test",
			Image:               "golang:1.27.3-alpine",
			Timeout:             time.Minute,
		},
	}
	db, err := sql.Open("sqlite", cfg.SQLite.Path)
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, db.Close()) })
	provider, err := goose.NewProvider(goose.DialectSQLite3, db, os.DirFS(dir), goose.WithDisableGlobalRegistry(true))
	require.NoError(t, err)
	_, err = provider.Up(t.Context())
	require.NoError(t, err)
	return cfg, db
}

func TestRuntimeReleasesJournalWhenDockerConstructionFails(t *testing.T) {
	cfg, _ := testConfig(t)
	t.Setenv("DOCKER_HOST", "invalid")
	err := run(t.Context(), cfg, zerolog.Nop())
	require.ErrorContains(t, err, "create pinn runner")
	journal, err := sqlite.New(cfg.SQLite)
	require.NoError(t, err, "startup failure must release the journal ownership lock")
	require.NoError(t, journal.Close())
}

func TestCommandConsumerRequiresAtLeastOnceSetup(t *testing.T) {
	t.Parallel()
	for _, topic := range []string{"tasks.run.queue", "tasks.predict.queue"} {
		_, err := newCommandConsumer(kafka.ConsumerConfig{}, topic, nil, zerolog.Nop())
		require.ErrorIs(t, err, kafka.ErrInvalidStrategySetup)
	}
}

func TestRuntimeRejectsInvalidWorkerConfigBeforeOpeningJournal(t *testing.T) {
	t.Parallel()
	cfg, _ := testConfig(t)
	cfg.Recovery.ActiveLimit = 0
	require.Error(t, run(context.Background(), cfg, zerolog.Nop()))
	journal, err := sqlite.New(cfg.SQLite)
	require.NoError(t, err)
	require.NoError(t, journal.Close())
}
