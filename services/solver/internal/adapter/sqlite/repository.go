package sqlite

import (
	"context"
	"errors"
	"os"
	"time"

	sq "github.com/Masterminds/squirrel"

	shared "github.com/AridanWarlock/pinnAutomizer/pkg/sqlite"
)

// SQLite is the query API consumed by this service adapter.
type SQLite interface {
	Getx(ctx context.Context, dst any, query shared.Sqlizer) error
	Selectx(ctx context.Context, dst any, query shared.Sqlizer) error
	Execx(ctx context.Context, query shared.Sqlizer) (shared.Tag, error)
	InTransaction(ctx context.Context, fn func(context.Context) error) error
	Close() error
}

type Repository struct {
	sqlite SQLite
	sb     sq.StatementBuilderType
}

var ErrAlreadyOwned = shared.ErrAlreadyOwned

func New(cfg Config) (_ *Repository, err error) {
	client, err := shared.New(shared.Config{Path: cfg.Path, BusyTimeout: cfg.BusyTimeout})
	if err != nil {
		return nil, err
	}
	ready := false
	defer func() {
		if !ready {
			err = errors.Join(err, client.Close())
		}
	}()
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err = client.CheckMigrations(ctx, os.DirFS(cfg.MigrationsDir)); err != nil {
		return nil, err
	}
	ready = true
	return &Repository{sqlite: client, sb: sq.StatementBuilder.PlaceholderFormat(sq.Question)}, nil
}

func (r *Repository) Close() error {
	return r.sqlite.Close()
}
