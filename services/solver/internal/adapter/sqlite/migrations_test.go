package sqlite

import (
	"context"
	"database/sql"
	"io/fs"
	"os"
	"path/filepath"
	"testing"
	"testing/fstest"
	"time"

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
	require.NoError(t, sqliteClient(repo).Database().QueryRow("SELECT COUNT(*) FROM goose_db_version WHERE version_id>0 AND is_applied=1").Scan(&count))
	require.Equal(t, 2, count)
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
	files["20990101000000_add_probe.sql"] = &fstest.MapFile{Data: []byte(`-- +goose Up
CREATE TABLE migration_probe(value TEXT NOT NULL);
INSERT INTO migration_probe(value) VALUES('once');
`)}
	require.ErrorContains(t, sqliteClient(repo).CheckMigrations(ctx, files), "pending")
	require.NoError(t, applyMigrations(ctx, sqliteClient(repo).Database(), files))
	require.NoError(t, applyMigrations(ctx, sqliteClient(repo).Database(), files))
	var count int
	require.NoError(t, sqliteClient(repo).Database().QueryRow("SELECT COUNT(*) FROM migration_probe").Scan(&count))
	require.Equal(t, 1, count)
	stored, err := repo.GetAttempt(ctx, a.RunID)
	require.NoError(t, err)
	require.True(t, stored.Matches(a))
	require.ErrorContains(t, sqliteClient(repo).CheckMigrations(ctx, os.DirFS(configDir(t))), "newer than migration files")

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
	files["20990101000000_failed.sql"] = &fstest.MapFile{Data: []byte(`-- +goose Up
CREATE TABLE partial_migration(value TEXT);
INSERT INTO missing_table(value) VALUES('failure');
`)}
	require.Error(t, applyMigrations(ctx, sqliteClient(repo).Database(), files))
	var count int
	require.NoError(t, sqliteClient(repo).Database().QueryRow("SELECT COUNT(*) FROM sqlite_master WHERE name='partial_migration'").Scan(&count))
	require.Zero(t, count)
	require.NoError(t, sqliteClient(repo).Database().QueryRow("SELECT COUNT(*) FROM goose_db_version WHERE version_id>0 AND is_applied=1").Scan(&count))
	require.Equal(t, 2, count)
	stored, err := repo.GetAttempt(ctx, a.RunID)
	require.NoError(t, err)
	require.True(t, stored.Matches(a))
	require.NoError(t, sqliteClient(repo).CheckMigrations(ctx, os.DirFS(configDir(t))))
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

func TestCleanupMigrationPreservesPreviouslyFinishedAttempts(t *testing.T) {
	cfg := Config{
		Path:          filepath.Join(t.TempDir(), "upgrade.db"),
		BusyTimeout:   50 * time.Millisecond,
		MigrationsDir: configDir(t),
	}
	db, err := sql.Open("sqlite", cfg.Path)
	require.NoError(t, err)
	defer db.Close()
	first := "20261014120000_create_solver_journal.sql"
	data, err := os.ReadFile(filepath.Join(cfg.MigrationsDir, first))
	require.NoError(t, err)
	require.NoError(t, applyMigrations(t.Context(), db, fstest.MapFS{
		first: &fstest.MapFile{Data: data},
	}))
	a := attempt(t)
	_, err = a.RecordContainer("retained-container")
	require.NoError(t, err)
	code := 0
	_, err = a.Finish(&code, nil, a.CreatedAt)
	require.NoError(t, err)
	row := newAttemptRow(a)
	_, err = db.Exec("INSERT INTO attempts(run_id,task_id,mode,state,container_id,created_at,completed_at,exit_code) VALUES(?,?,?,?,?,?,?,?)",
		row.RunID, row.TaskID, row.Mode, row.State, row.ContainerID, row.CreatedAt, row.CompletedAt, row.ExitCode)
	require.NoError(t, err)
	require.NoError(t, applyMigrations(t.Context(), db, os.DirFS(cfg.MigrationsDir)))
	require.NoError(t, db.Close())
	repo := open(t, cfg)
	restored, err := repo.GetAttempt(t.Context(), a.RunID)
	require.NoError(t, err)
	require.Equal(t, a, restored)
	require.True(t, restored.NeedsCleanup(), "old retained containers must enter recovery after upgrade")
}
