package sqlite

import (
	"context"
	"testing"
	"testing/fstest"

	"github.com/pressly/goose/v3"
	"github.com/stretchr/testify/require"
)

func TestCheckMigrationsRejectsMissingAppliedSource(t *testing.T) {
	client := queryClient(t)
	ctx := context.Background()
	files := fstest.MapFS{
		"00001_first.sql":  {Data: []byte("-- +goose Up\nCREATE TABLE first_probe(id INTEGER);\n")},
		"00002_second.sql": {Data: []byte("-- +goose Up\nCREATE TABLE second_probe(id INTEGER);\n")},
		"00003_third.sql":  {Data: []byte("-- +goose Up\nCREATE TABLE third_probe(id INTEGER);\n")},
	}
	provider, err := goose.NewProvider(goose.DialectSQLite3, client.Database(), files, goose.WithDisableGlobalRegistry(true))
	require.NoError(t, err)
	_, err = provider.Up(ctx)
	require.NoError(t, err)
	require.NoError(t, client.CheckMigrations(ctx, files))
	delete(files, "00002_second.sql")
	require.ErrorContains(t, client.CheckMigrations(ctx, files), "missing applied migration 2")

	// A readiness check must not alter the existing schema or migration history.
	var count int
	require.NoError(t, client.Database().QueryRow("SELECT COUNT(*) FROM goose_db_version WHERE version_id > 0 AND is_applied = 1").Scan(&count))
	require.Equal(t, 3, count)
	require.NoError(t, client.Database().QueryRow("SELECT COUNT(*) FROM second_probe").Scan(&count))
}
