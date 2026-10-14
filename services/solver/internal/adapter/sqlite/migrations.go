package sqlite

import (
	"context"
	"fmt"
	"io/fs"

	"github.com/pressly/goose/v3"

	"github.com/AridanWarlock/pinnAutomizer/solver/migrations"
)

func (r *Repository) initializeSchema(ctx context.Context) error {
	return r.applyMigrations(ctx, migrations.Files)
}

func (r *Repository) applyMigrations(ctx context.Context, files fs.FS) error {
	provider, err := goose.NewProvider(goose.DialectSQLite3, r.db, files, goose.WithDisableGlobalRegistry(true))
	if err != nil {
		return fmt.Errorf("create SQLite migration provider: %w", err)
	}
	// Reject a database created by a newer solver before touching application tables.
	// The supported version is derived from migration files, not a source constant.
	version, err := provider.GetDBVersion(ctx)
	if err != nil {
		return fmt.Errorf("read SQLite migration version: %w", err)
	}
	sources := provider.ListSources()
	if version > sources[len(sources)-1].Version {
		return fmt.Errorf("unsupported solver journal schema: database migration %d is newer than this binary", version)
	}
	if _, err = provider.Up(ctx); err != nil {
		return fmt.Errorf("migrate SQLite journal: %w", err)
	}
	return nil
}
