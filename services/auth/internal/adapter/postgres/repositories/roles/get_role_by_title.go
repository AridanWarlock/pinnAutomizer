package roles

import (
	"context"

	. "github.com/AridanWarlock/pinnAutomizer/auth/internal/adapter/postgres/schema"
	"github.com/AridanWarlock/pinnAutomizer/pkg/core"
	. "github.com/Masterminds/squirrel"
)

func (r *Repository) GetRoleByTitle(ctx context.Context, title string) (core.Role, error) {
	query := r.sb.
		Select(RolesTableColumns...).
		From(RolesTable).
		Where(Eq{RolesTableColumnTitle: title})

	var outRow RoleRaw
	if err := r.pool.Getx(ctx, &outRow, query); err != nil {
		return core.Role{}, err
	}
	return ToModel(outRow), nil
}
