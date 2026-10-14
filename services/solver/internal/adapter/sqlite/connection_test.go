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
	_, err := repo.db.Exec("INSERT INTO goose_db_version(version_id,is_applied) VALUES(99999999999999,1)")
	require.NoError(t, err)
	require.NoError(t, repo.Close())
	_, err = New(cfg)
	require.ErrorContains(t, err, "unsupported solver journal schema")
	db, err := sql.Open("sqlite", cfg.Path)
	require.NoError(t, err)
	defer db.Close()
	_, err = db.Exec("DELETE FROM goose_db_version WHERE version_id=99999999999999")
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

func TestUnversionedJournalIsNotSilentlyBaselined(t *testing.T) {
	cfg := config(t)
	db, err := sql.Open("sqlite", cfg.Path)
	require.NoError(t, err)
	defer db.Close()
	_, err = db.Exec("CREATE TABLE attempts(marker TEXT); INSERT INTO attempts(marker) VALUES('retained')")
	require.NoError(t, err)
	_, err = New(cfg)
	require.ErrorContains(t, err, "migrate SQLite journal")
	var marker string
	require.NoError(t, db.QueryRow("SELECT marker FROM attempts").Scan(&marker))
	require.Equal(t, "retained", marker)
	var count int
	require.NoError(t, db.QueryRow("SELECT COUNT(*) FROM goose_db_version WHERE version_id>0 AND is_applied=1").Scan(&count))
	require.Zero(t, count)
}
