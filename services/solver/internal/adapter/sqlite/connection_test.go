package sqlite

import (
	"database/sql"
	"os"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestFailedJournalFileCreationReleasesOwner(t *testing.T) {
	cfg := config(t)
	require.NoError(t, os.Mkdir(cfg.Path, 0700))
	_, err := New(cfg)
	require.Error(t, err)
	require.NotErrorIs(t, err, ErrAlreadyOwned)
	require.NoError(t, os.Remove(cfg.Path))
	open(t, cfg)
}

func TestUnsupportedSchemaReleasesOwner(t *testing.T) {
	cfg := config(t)
	repo := open(t, cfg)
	_, err := repo.db.Exec("PRAGMA user_version=99")
	require.NoError(t, err)
	require.NoError(t, repo.Close())
	_, err = New(cfg)
	require.ErrorContains(t, err, "unsupported solver journal schema")
	db, err := sql.Open("sqlite", cfg.Path)
	require.NoError(t, err)
	defer db.Close()
	_, err = db.Exec("PRAGMA user_version=1")
	require.NoError(t, err)
	open(t, cfg)
}

func TestFailedSchemaCreationRollsBackAndReleasesOwner(t *testing.T) {
	cfg := config(t)
	db, err := sql.Open("sqlite", cfg.Path)
	require.NoError(t, err)
	defer db.Close()
	// The second CREATE fails after the first CREATE succeeded in the transaction.
	_, err = db.Exec("CREATE TABLE outbox(dummy TEXT)")
	require.NoError(t, err)
	_, err = New(cfg)
	require.Error(t, err)
	var count int
	require.NoError(t, db.QueryRow("SELECT COUNT(*) FROM sqlite_master WHERE name='attempts'").Scan(&count))
	require.Zero(t, count)
	_, err = db.Exec("DROP TABLE outbox")
	require.NoError(t, err)
	open(t, cfg)
}
