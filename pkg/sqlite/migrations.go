package sqlite

import (
	"context"
	"fmt"
	"io/fs"

	"github.com/pressly/goose/v3"
)

// CheckMigrations only verifies the externally applied schema; it never runs Up.
func (s *SQLite) CheckMigrations(ctx context.Context, files fs.FS) error {
	var present int64
	if err := s.db.QueryRowContext(ctx, "SELECT version_id FROM goose_db_version ORDER BY id DESC LIMIT 1").Scan(&present); err != nil {
		return fmt.Errorf("SQLite schema is not migrated; run goose up: %w", err)
	}
	provider, err := goose.NewProvider(goose.DialectSQLite3, s.db, files, goose.WithDisableGlobalRegistry(true))
	if err != nil {
		return fmt.Errorf("read SQLite migrations: %w", err)
	}
	version, err := provider.GetDBVersion(ctx)
	if err != nil {
		return err
	}
	sources := provider.ListSources()
	if version > sources[len(sources)-1].Version {
		return fmt.Errorf("unsupported SQLite schema: database migration %d is newer than migration files", version)
	}
	statuses, err := provider.Status(ctx)
	if err != nil {
		return err
	}
	for _, status := range statuses {
		if status.State == goose.StatePending {
			return fmt.Errorf("SQLite migration %s is pending; run goose up", status.Source.Path)
		}
	}
	return nil
}
