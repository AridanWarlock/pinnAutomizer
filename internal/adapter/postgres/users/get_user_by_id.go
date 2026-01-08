package users

import (
	"context"
	"errors"
	"pinnAutomizer/internal/domain"

	sq "github.com/Masterminds/squirrel"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

func (r *UsersRepository) GetUserByID(ctx context.Context, id uuid.UUID) (*domain.User, error) {
	query := r.sb.
		Select(usersTableColumns...).
		From(usersTable).
		Where(sq.Eq{usersTableColumnID: id})

	var row UserRow
	if err := r.pool.Getx(ctx, &row, query); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, err
	}
	return ToModel(&row), nil
}
