package sqlite

import (
	"database/sql"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestLifecycleAndNoSchemaCreation(t *testing.T) {
	path := filepath.Join(t.TempDir(), "journal.db")
	db, err := sql.Open("sqlite", path)
	require.NoError(t, err)
	_, err = db.Exec("CREATE TABLE probe(value TEXT)")
	require.NoError(t, err)
	require.NoError(t, db.Close())
	cfg := Config{Path: path, BusyTimeout: time.Second}
	client, err := New(cfg)
	require.NoError(t, err)
	defer client.Close()
	_, err = New(cfg)
	require.ErrorIs(t, err, ErrAlreadyOwned)
	var count int
	require.NoError(t, client.Database().QueryRow("SELECT COUNT(*) FROM sqlite_master WHERE name='goose_db_version'").Scan(&count))
	require.Zero(t, count)
	require.NoError(t, client.Close())
	client, err = New(cfg)
	require.NoError(t, err)
	require.NoError(t, client.Close())
}

func TestMissingDatabaseReleasesLock(t *testing.T) {
	path := filepath.Join(t.TempDir(), "missing.db")
	cfg := Config{Path: path, BusyTimeout: time.Second}
	_, err := New(cfg)
	require.Error(t, err)
	_, err = os.Stat(path)
	require.True(t, os.IsNotExist(err))
	file, err := os.Create(path)
	require.NoError(t, err)
	require.NoError(t, file.Close())
	client, err := New(cfg)
	require.NoError(t, err)
	require.NoError(t, client.Close())
}
