package sqlite

import (
	"context"
	"io/fs"
	"testing"
	"testing/fstest"

	"github.com/pressly/goose/v3"
	"github.com/stretchr/testify/require"

	"github.com/AridanWarlock/pinnAutomizer/solver/migrations"
)

func TestMigrationsReopenDoesNotReapplyAndPreservesData(t *testing.T) {
	cfg := config(t)
	repo := open(t, cfg)
	ctx := context.Background()
	a := attempt(t)
	_, _, err := repo.RegisterAttempt(ctx, a)
	require.NoError(t, err)
	require.NoError(t, repo.Close())
	repo = open(t, cfg)
	stored, err := repo.GetAttempt(ctx, a.RunID)
	require.NoError(t, err)
	require.True(t, stored.Matches(a))
	var count int
	require.NoError(t, repo.db.QueryRow("SELECT COUNT(*) FROM goose_db_version WHERE version_id>0 AND is_applied=1").Scan(&count))
	require.Equal(t, 1, count)
}

func TestPendingMigrationRunsOnceAndCanBeRolledBack(t *testing.T) {
	repo := open(t, config(t))
	ctx := context.Background()
	a := attempt(t)
	_, _, err := repo.RegisterAttempt(ctx, a)
	require.NoError(t, err)
	files := fstest.MapFS{}
	entries, err := fs.ReadDir(migrations.Files, ".")
	require.NoError(t, err)
	for _, entry := range entries {
		data, err := fs.ReadFile(migrations.Files, entry.Name())
		require.NoError(t, err)
		files[entry.Name()] = &fstest.MapFile{Data: data}
	}
	files["20261014130000_add_probe.sql"] = &fstest.MapFile{Data: []byte(`-- +goose Up
CREATE TABLE migration_probe(value TEXT NOT NULL);
INSERT INTO migration_probe(value) VALUES('once');
-- +goose Down
DROP TABLE migration_probe;
`)}
	require.NoError(t, repo.applyMigrations(ctx, files))
	require.NoError(t, repo.applyMigrations(ctx, files))
	var count int
	require.NoError(t, repo.db.QueryRow("SELECT COUNT(*) FROM migration_probe").Scan(&count))
	require.Equal(t, 1, count)
	stored, err := repo.GetAttempt(ctx, a.RunID)
	require.NoError(t, err)
	require.True(t, stored.Matches(a))
	require.ErrorContains(t, repo.applyMigrations(ctx, migrations.Files), "newer than this binary")
	provider, err := goose.NewProvider(goose.DialectSQLite3, repo.db, files, goose.WithDisableGlobalRegistry(true))
	require.NoError(t, err)
	_, err = provider.Down(ctx)
	require.NoError(t, err)
	require.NoError(t, repo.initializeSchema(ctx))
	stored, err = repo.GetAttempt(ctx, a.RunID)
	require.NoError(t, err)
	require.True(t, stored.Matches(a))
	// Down of the initial migration is tested only on this disposable database.
	_, err = provider.Down(ctx)
	require.NoError(t, err)
	require.NoError(t, repo.db.QueryRow("SELECT COUNT(*) FROM sqlite_master WHERE name IN ('attempts','outbox')").Scan(&count))
	require.Zero(t, count)
}

func TestFailedPendingMigrationKeepsDataAndVersion(t *testing.T) {
	repo := open(t, config(t))
	ctx := context.Background()
	a := attempt(t)
	_, _, err := repo.RegisterAttempt(ctx, a)
	require.NoError(t, err)
	entries, err := fs.ReadDir(migrations.Files, ".")
	require.NoError(t, err)
	files := fstest.MapFS{}
	for _, entry := range entries {
		data, err := fs.ReadFile(migrations.Files, entry.Name())
		require.NoError(t, err)
		files[entry.Name()] = &fstest.MapFile{Data: data}
	}
	files["20261014130000_failed.sql"] = &fstest.MapFile{Data: []byte(`-- +goose Up
CREATE TABLE partial_migration(value TEXT);
INSERT INTO missing_table(value) VALUES('failure');
-- +goose Down
DROP TABLE partial_migration;
`)}
	require.Error(t, repo.applyMigrations(ctx, files))
	var count int
	require.NoError(t, repo.db.QueryRow("SELECT COUNT(*) FROM sqlite_master WHERE name='partial_migration'").Scan(&count))
	require.Zero(t, count)
	require.NoError(t, repo.db.QueryRow("SELECT COUNT(*) FROM goose_db_version WHERE version_id>0 AND is_applied=1").Scan(&count))
	require.Equal(t, 1, count)
	stored, err := repo.GetAttempt(ctx, a.RunID)
	require.NoError(t, err)
	require.True(t, stored.Matches(a))
	require.NoError(t, repo.initializeSchema(ctx))
}
