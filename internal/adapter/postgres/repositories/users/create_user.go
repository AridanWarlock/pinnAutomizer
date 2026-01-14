package users

import (
	"context"
	. "pinnAutomizer/internal/adapter/postgres/schema"
	"pinnAutomizer/internal/domain"
	"strings"
)

func (r *Repository) CreateUser(ctx context.Context, user domain.User) (domain.User, error) {
	row := FromModel(user)
	query := r.sb.
		Insert(UsersTable).
		Columns(UsersTableColumns...).
		Values(row.Values()...).
		Suffix("RETURNING " + strings.Join(UsersTableColumns, ","))

	var outRow UserRow
	if err := r.pool.Getx(ctx, &outRow, query); err != nil {
		return domain.User{}, err
	}
	return ToModel(outRow), nil
}
