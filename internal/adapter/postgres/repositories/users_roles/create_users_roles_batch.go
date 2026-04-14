package users_roles

import (
	"context"
	"fmt"
	"strings"

	. "github.com/AridanWarlock/pinnAutomizer/internal/adapter/postgres/schema"
	"github.com/AridanWarlock/pinnAutomizer/internal/domain"
	"github.com/AridanWarlock/pinnAutomizer/pkg/errs"
)

func (r *Repository) CreateUsersRolesBatch(ctx context.Context, usersRoles []domain.UsersRoles) ([]domain.UsersRoles, error) {
	batchSize := len(usersRoles)
	if batchSize == 0 || batchSize > 100 {
		return nil, fmt.Errorf(
			"%w: invalid batch size=%d",
			errs.ErrInvalidArgument,
			batchSize,
		)
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
