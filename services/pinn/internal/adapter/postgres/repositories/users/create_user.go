package users

import (
	"context"
	"strings"

	. "github.com/AridanWarlock/pinnAutomizer/pinn/internal/adapter/postgres/schema"
	"github.com/AridanWarlock/pinnAutomizer/pinn/internal/domain"
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
