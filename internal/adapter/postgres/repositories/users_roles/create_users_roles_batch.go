package users_roles

import (
	"context"
	. "pinnAutomizer/internal/adapter/postgres/pg_errors"
	. "pinnAutomizer/internal/adapter/postgres/schema"
	"pinnAutomizer/internal/domain"
	"strings"
)

func (r *Repository) CreateUsersRolesBatch(ctx context.Context, usersRoles []domain.UsersRoles) ([]domain.UsersRoles, error) {
	batchSize := len(usersRoles)
	if batchSize == 0 || batchSize > 100 {
		return nil, ErrInvalidBatchSize
	}

	query := r.sb.
		Insert(UsersRolesTable).
		Columns(UsersRolesTableColumns...)

	for _, ur := range usersRoles {
		values := FromModel(ur).Values()
		query = query.Values(values...)
	}

	query = query.Suffix("RETURNING " + strings.Join(UsersRolesTableColumns, ","))

	var outRows []UsersRolesRow
	if err := r.pool.Selectx(ctx, &outRows, query); err != nil {
		return nil, err
	}

	res := make([]domain.UsersRoles, batchSize)
	for i, row := range outRows {
		res[i] = ToModel(row)
	}
	return res, nil
}
