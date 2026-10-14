package sqlite

import (
	"database/sql"
	"os"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestFailedJournalFileCreationReleasesOwner(t *testing.T) {
	cfg := config(t)
	require.NoError(t, os.Mkdir(cfg.Path+".directory", 0700))
	bad := cfg
	bad.Path += ".directory"
	_, err := New(bad)
	require.Error(t, err)
	require.NotErrorIs(t, err, ErrAlreadyOwned)
	require.NoError(t, os.Remove(cfg.Path+".directory"))
	open(t, cfg)
}

func TestUnsupportedSchemaReleasesOwner(t *testing.T) {
	cfg := config(t)
	repo := open(t, cfg)
	_, err := repo.db.Exec("INSERT INTO goose_db_version(version_id,is_applied) VALUES(99999999999999,1)")
	require.NoError(t, err)
	require.NoError(t, repo.Close())
	_, err = New(cfg)
	require.ErrorContains(t, err, "unsupported SQLite schema")
	db, err := sql.Open("sqlite", cfg.Path)
	require.NoError(t, err)
	defer db.Close()
	_, err = db.Exec("DELETE FROM goose_db_version WHERE version_id=99999999999999")
	require.NoError(t, err)
	open(t, cfg)
}

func TestMissingDatabaseIsNotCreated(t *testing.T) {
	cfg := config(t)
	cfg.Path += ".missing"
	_, err := New(cfg)
	require.Error(t, err)
	_, err = os.Stat(cfg.Path)
	require.True(t, os.IsNotExist(err))
}

func TestUnversionedJournalIsNotSilentlyBaselined(t *testing.T) {
	cfg := config(t)
	cfg.Path += ".legacy"
	db, err := sql.Open("sqlite", cfg.Path)
	require.NoError(t, err)
	defer db.Close()
	_, err = db.Exec("CREATE TABLE attempts(marker TEXT); INSERT INTO attempts VALUES('retained')")
	require.NoError(t, err)
	_, err = New(cfg)
	require.ErrorContains(t, err, "not migrated")
	var marker string
	require.NoError(t, db.QueryRow("SELECT marker FROM attempts").Scan(&marker))
	require.Equal(t, "retained", marker)
	var count int
	require.NoError(t, db.QueryRow("SELECT COUNT(*) FROM sqlite_master WHERE name='goose_db_version'").Scan(&count))
	require.Zero(t, count)
}
