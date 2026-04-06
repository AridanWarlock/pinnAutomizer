package users

import (
	"context"
	"fmt"

	"github.com/AridanWarlock/pinnAutomizer/internal/adapter/postgres/pgerr"
	. "github.com/AridanWarlock/pinnAutomizer/internal/adapter/postgres/schema"
	"github.com/AridanWarlock/pinnAutomizer/internal/domain"
	"github.com/AridanWarlock/pinnAutomizer/internal/errs"

	. "github.com/Masterminds/squirrel"
)

func (r *Repository) GetUserByLogin(ctx context.Context, login string) (domain.User, error) {
	query := r.sb.
		Select(UsersTableColumns...).
		From(UsersTable).
		Where(Eq{UsersTableColumnLogin: login})

	var outRow UserRow
	if err := r.pool.Getx(ctx, &outRow, query); err != nil {
		if pgerr.IsNotFound(err) {
			return domain.User{}, fmt.Errorf(
				"user with login=%s: %w",
				login,
				errs.ErrNotFound,
			)
		}
		return domain.User{}, pgerr.ScanErr(err)
	}

	return ToModel(outRow), nil
}
