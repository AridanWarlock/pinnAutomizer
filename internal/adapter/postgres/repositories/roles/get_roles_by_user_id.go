package roles

import (
	"context"
	"fmt"
	. "pinnAutomizer/internal/adapter/postgres/schema"
	"pinnAutomizer/internal/domain"

	sq "github.com/Masterminds/squirrel"
	"github.com/google/uuid"
)

func (r *Repository) GetRolesByUserID(ctx context.Context, userID uuid.UUID) ([]domain.Role, error) {
	q := r.sb.Select(fmt.Sprintf("r.%s, r.%s", RolesTableColumnID, RolesTableColumnTitle)).
		From(UsersRolesTable + " ur").
		InnerJoin(fmt.Sprintf("%s r on r.%s = ur.%s", RolesTable, RolesTableColumnID, UsersRolesTableColumnRoleID)).
		Where(sq.Eq{"u." + UsersTableColumnID: userID})

	var rows []RoleRaw
	if err := r.pool.Selectx(ctx, &rows, q); err != nil {
		return nil, err
	}

	roles := make([]domain.Role, len(rows))
	for i, row := range rows {
		roles[i] = ToModel(row)
	}
	return roles, nil
}
