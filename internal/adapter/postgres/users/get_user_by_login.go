package users

import (
	"context"
	"errors"
	"pinnAutomizer/internal/domain"

	sq "github.com/Masterminds/squirrel"
	"github.com/jackc/pgx/v5"
)

func (r *UsersRepository) GetUserByLogin(ctx context.Context, login string) (*domain.User, error) {
	query := r.sb.
		Select(usersTableColumns...).
		From(usersTable).
		Where(sq.Eq{usersTableColumnLogin: login})

	var outRow UserRow
	if err := r.pool.Getx(ctx, &outRow, query); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, err
	}

	return ToModel(&outRow), nil
}
