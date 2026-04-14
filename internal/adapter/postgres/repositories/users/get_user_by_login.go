package users

import (
	"context"

	. "github.com/AridanWarlock/pinnAutomizer/internal/adapter/postgres/schema"
	"github.com/AridanWarlock/pinnAutomizer/internal/domain"

	. "github.com/Masterminds/squirrel"
)

func (r *Repository) GetUserByLogin(ctx context.Context, login string) (domain.User, error) {
	query := r.sb.
		Select(UsersTableColumns...).
		From(UsersTable).
		Where(Eq{UsersTableColumnLogin: login})

	var outRow UserRow
	if err := r.pool.Getx(ctx, &outRow, query); err != nil {
		return domain.User{}, err
	}

	return ToModel(outRow), nil
}
