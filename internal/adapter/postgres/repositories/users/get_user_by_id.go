package users

import (
	"context"
	"errors"
	"pinnAutomizer/internal/adapter/postgres/pg_errors"
	"pinnAutomizer/internal/adapter/postgres/schema"
	"pinnAutomizer/internal/domain"

	sq "github.com/Masterminds/squirrel"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

func (r *Repository) GetUserByID(ctx context.Context, id uuid.UUID) (*domain.User, error) {
	query := r.sb.
		Select(schema.UsersTableColumns...).
		From(schema.UsersTable).
		Where(sq.Eq{schema.UsersTableColumnID: id})

	var row UserRow
	if err := r.pool.Getx(ctx, &row, query); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, pg_errors.ErrNotFound
		}
		return nil, err
	}
	return ToModel(&row), nil
}
