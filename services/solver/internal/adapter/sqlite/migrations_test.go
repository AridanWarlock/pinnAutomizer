package sqlite

import (
	"context"
	"database/sql"
	"io/fs"
	"os"
	"path/filepath"
	"testing"
	"testing/fstest"

	"github.com/pressly/goose/v3"
	"github.com/stretchr/testify/require"
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
	require.NoError(t, repo.sqlite.Database().QueryRow("SELECT COUNT(*) FROM goose_db_version WHERE version_id>0 AND is_applied=1").Scan(&count))
	require.Equal(t, 1, count)
}

func TestPendingMigrationRunsOnce(t *testing.T) {
	repo := open(t, config(t))
	ctx := context.Background()
	a := attempt(t)
	_, _, err := repo.RegisterAttempt(ctx, a)
	require.NoError(t, err)
	files := fstest.MapFS{}
	entries, err := fs.ReadDir(os.DirFS(configDir(t)), ".")
	require.NoError(t, err)
	for _, entry := range entries {
		data, err := fs.ReadFile(os.DirFS(configDir(t)), entry.Name())
		require.NoError(t, err)
		files[entry.Name()] = &fstest.MapFile{Data: data}
	}
	files["20261014130000_add_probe.sql"] = &fstest.MapFile{Data: []byte(`-- +goose Up
CREATE TABLE migration_probe(value TEXT NOT NULL);
INSERT INTO migration_probe(value) VALUES('once');
`)}
	require.ErrorContains(t, repo.sqlite.CheckMigrations(ctx, files), "pending")
	require.NoError(t, applyMigrations(ctx, repo.sqlite.Database(), files))
	require.NoError(t, applyMigrations(ctx, repo.sqlite.Database(), files))
	var count int
	require.NoError(t, repo.sqlite.Database().QueryRow("SELECT COUNT(*) FROM migration_probe").Scan(&count))
	require.Equal(t, 1, count)
	stored, err := repo.GetAttempt(ctx, a.RunID)
	require.NoError(t, err)
	require.True(t, stored.Matches(a))
	require.ErrorContains(t, repo.sqlite.CheckMigrations(ctx, os.DirFS(configDir(t))), "newer than migration files")

}

func TestFailedPendingMigrationKeepsDataAndVersion(t *testing.T) {
	repo := open(t, config(t))
	ctx := context.Background()
	a := attempt(t)
	_, _, err := repo.RegisterAttempt(ctx, a)
	require.NoError(t, err)
	entries, err := fs.ReadDir(os.DirFS(configDir(t)), ".")
	require.NoError(t, err)
	files := fstest.MapFS{}
	for _, entry := range entries {
		data, err := fs.ReadFile(os.DirFS(configDir(t)), entry.Name())
		require.NoError(t, err)
		files[entry.Name()] = &fstest.MapFile{Data: data}
	}
	files["20261014130000_failed.sql"] = &fstest.MapFile{Data: []byte(`-- +goose Up
CREATE TABLE partial_migration(value TEXT);
INSERT INTO missing_table(value) VALUES('failure');
`)}
	require.Error(t, applyMigrations(ctx, repo.sqlite.Database(), files))
	var count int
	require.NoError(t, repo.sqlite.Database().QueryRow("SELECT COUNT(*) FROM sqlite_master WHERE name='partial_migration'").Scan(&count))
	require.Zero(t, count)
	require.NoError(t, repo.sqlite.Database().QueryRow("SELECT COUNT(*) FROM goose_db_version WHERE version_id>0 AND is_applied=1").Scan(&count))
	require.Equal(t, 1, count)
	stored, err := repo.GetAttempt(ctx, a.RunID)
	require.NoError(t, err)
	require.True(t, stored.Matches(a))
	require.NoError(t, repo.sqlite.CheckMigrations(ctx, os.DirFS(configDir(t))))
}

func configDir(t *testing.T) string {
	t.Helper()
	dir, err := filepath.Abs("../../../migrations")
	require.NoError(t, err)
	return dir
}

func applyMigrations(ctx context.Context, db *sql.DB, files fs.FS) error {
	provider, err := goose.NewProvider(goose.DialectSQLite3, db, files, goose.WithDisableGlobalRegistry(true))
	if err != nil {
		return err
	}
	_, err = provider.Up(ctx)
	return err
}

func migrate(t *testing.T, path string, files fs.FS) {
	t.Helper()
	db, err := sql.Open("sqlite", path)
	require.NoError(t, err)
	defer db.Close()
	require.NoError(t, applyMigrations(context.Background(), db, files))
}
