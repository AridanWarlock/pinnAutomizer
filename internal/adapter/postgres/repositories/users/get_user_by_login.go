package users

import (
	"context"
	"errors"

	. "github.com/AridanWarlock/pinnAutomizer/internal/adapter/postgres/pg_errors"
	. "github.com/AridanWarlock/pinnAutomizer/internal/adapter/postgres/schema"
	"github.com/AridanWarlock/pinnAutomizer/internal/domain"

	. "github.com/Masterminds/squirrel"
	"github.com/jackc/pgx/v5"
)

func (r *Repository) GetUserByLogin(ctx context.Context, login string) (domain.User, error) {
	query := r.sb.
		Select(UsersTableColumns...).
		From(UsersTable).
		Where(Eq{UsersTableColumnLogin: login})

	var outRow UserRow
	if err := r.pool.Getx(ctx, &outRow, query); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return domain.User{}, ErrNotFound
		}
		return domain.User{}, err
	}

	return ToModel(outRow), nil
}
