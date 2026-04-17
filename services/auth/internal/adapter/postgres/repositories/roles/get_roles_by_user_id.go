package roles

import (
	"context"
	"fmt"

	. "github.com/AridanWarlock/pinnAutomizer/auth/internal/adapter/postgres/schema"
	"github.com/AridanWarlock/pinnAutomizer/pkg/core"

	sq "github.com/Masterminds/squirrel"
	"github.com/google/uuid"
)

func (r *Repository) GetRolesByUserID(ctx context.Context, userID uuid.UUID) ([]core.Role, error) {
	q := r.sb.Select(fmt.Sprintf("r.%s, r.%s", RolesTableColumnID, RolesTableColumnTitle)).
		From(UsersRolesTable + " ur").
		InnerJoin(fmt.Sprintf("%s r on r.%s = ur.%s", RolesTable, RolesTableColumnID, UsersRolesTableColumnRoleID)).
		Where(sq.Eq{"ur." + UsersRolesTableColumnUserID: userID})

	var rows []RoleRaw
	if err := r.pool.Selectx(ctx, &rows, q); err != nil {
		return nil, err
	}

	roles := make([]core.Role, len(rows))
	for i, row := range rows {
		roles[i] = ToModel(row)
	}
	return roles, nil
}
