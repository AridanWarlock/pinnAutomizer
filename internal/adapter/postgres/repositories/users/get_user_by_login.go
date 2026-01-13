package users

import (
	"context"
	"errors"
	"pinnAutomizer/internal/adapter/postgres/pg_errors"
	"pinnAutomizer/internal/adapter/postgres/schema"
	"pinnAutomizer/internal/domain"

	sq "github.com/Masterminds/squirrel"
	"github.com/jackc/pgx/v5"
)

func (r *Repository) GetUserByLogin(ctx context.Context, login string) (*domain.User, error) {
	query := r.sb.
		Select(schema.UsersTableColumns...).
		From(schema.UsersTable).
		Where(sq.Eq{schema.UsersTableColumnLogin: login})

	var outRow UserRow
	if err := r.pool.Getx(ctx, &outRow, query); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, pg_errors.ErrNotFound
		}
		return nil, err
	}

	return ToModel(&outRow), nil
}
