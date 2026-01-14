package roles

import (
	"context"
	"pinnAutomizer/internal/adapter/postgres/schema"
	"pinnAutomizer/internal/domain"

	sq "github.com/Masterminds/squirrel"
)

func (r *Repository) GetRoleByTitle(ctx context.Context, title string) (domain.Role, error) {
	query := r.sb.
		Select(schema.RolesTableColumns...).
		From(schema.RolesTable).
		Where(sq.Eq{schema.RolesTableColumnTitle: title})

	var outRow RoleRaw
	if err := r.pool.Getx(ctx, &outRow, query); err != nil {
		return domain.Role{}, err
	}
	return ToModel(outRow), nil
}
