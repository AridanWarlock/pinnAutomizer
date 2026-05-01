package postgres

import (
	"context"
	"fmt"

	"github.com/AridanWarlock/pinnAutomizer/pkg/core"
	sq "github.com/Masterminds/squirrel"
	"github.com/google/uuid"
)

func (r *Repository) GetRoleByTitle(ctx context.Context, title string) (core.Role, error) {
	query := r.sb.
		Select(RolesColumns...).
		From(RolesTable).
		Where(sq.Eq{RolesTitle: title})

	var outRow RoleRaw
	if err := r.pool.Getx(ctx, &outRow, query); err != nil {
		return core.Role{}, err
	}
	return ToRoleModel(outRow), nil
}

func (r *Repository) GetRolesByUserID(ctx context.Context, userID uuid.UUID) ([]core.Role, error) {
	q := r.sb.Select(fmt.Sprintf("r.%s, r.%s", RolesID, RolesTitle)).
		From(UsersRolesTable + " ur").
		InnerJoin(fmt.Sprintf("%s r on r.%s = ur.%s", RolesTable, RolesID, UsersRolesRoleID)).
		Where(sq.Eq{"ur." + UsersRolesUserID: userID})

	var rows []RoleRaw
	if err := r.pool.Selectx(ctx, &rows, q); err != nil {
		return nil, err
	}

	roles := make([]core.Role, len(rows))
	for i, row := range rows {
		roles[i] = ToRoleModel(row)
	}
	return roles, nil
}
