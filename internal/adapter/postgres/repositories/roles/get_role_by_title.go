package roles

import (
	"context"
	"fmt"

	"github.com/AridanWarlock/pinnAutomizer/internal/adapter/postgres/pgerr"
	. "github.com/AridanWarlock/pinnAutomizer/internal/adapter/postgres/schema"
	"github.com/AridanWarlock/pinnAutomizer/internal/domain"
	"github.com/AridanWarlock/pinnAutomizer/internal/errs"

	. "github.com/Masterminds/squirrel"
)

func (r *Repository) GetRoleByTitle(ctx context.Context, title string) (domain.Role, error) {
	query := r.sb.
		Select(RolesTableColumns...).
		From(RolesTable).
		Where(Eq{RolesTableColumnTitle: title})

	var outRow RoleRaw
	if err := r.pool.Getx(ctx, &outRow, query); err != nil {
		if pgerr.IsNotFound(err) {
			return domain.Role{}, fmt.Errorf(
				"role with title=%s: %w",
				title,
				errs.ErrNotFound,
			)
		}
		return domain.Role{}, pgerr.ScanErr(err)
	}
	return ToModel(outRow), nil
}
