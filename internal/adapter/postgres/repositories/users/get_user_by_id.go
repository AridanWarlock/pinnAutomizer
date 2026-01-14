package users

import (
	"context"
	"errors"
	. "pinnAutomizer/internal/adapter/postgres/pg_errors"
	. "pinnAutomizer/internal/adapter/postgres/schema"
	"pinnAutomizer/internal/domain"

	. "github.com/Masterminds/squirrel"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

func (r *Repository) GetUserByID(ctx context.Context, id uuid.UUID) (domain.User, error) {
	query := r.sb.
		Select(UsersTableColumns...).
		From(UsersTable).
		Where(Eq{UsersTableColumnID: id})

	var row UserRow
	if err := r.pool.Getx(ctx, &row, query); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return domain.User{}, ErrNotFound
		}
		return domain.User{}, err
	}
	return ToModel(row), nil
}
