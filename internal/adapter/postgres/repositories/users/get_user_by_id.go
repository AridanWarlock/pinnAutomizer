package users

import (
	"context"
	"fmt"

	"github.com/AridanWarlock/pinnAutomizer/internal/adapter/postgres/pgerr"
	. "github.com/AridanWarlock/pinnAutomizer/internal/adapter/postgres/schema"
	"github.com/AridanWarlock/pinnAutomizer/internal/domain"
	"github.com/AridanWarlock/pinnAutomizer/internal/errs"

	. "github.com/Masterminds/squirrel"
	"github.com/google/uuid"
)

func (r *Repository) GetUserByID(ctx context.Context, id uuid.UUID) (domain.User, error) {
	query := r.sb.
		Select(UsersTableColumns...).
		From(UsersTable).
		Where(Eq{UsersTableColumnID: id})

	var row UserRow
	if err := r.pool.Getx(ctx, &row, query); err != nil {
		if pgerr.IsNotFound(err) {
			return domain.User{}, fmt.Errorf(
				"user with id=%v: %w",
				id,
				errs.ErrNotFound,
			)
		}
		return domain.User{}, pgerr.ScanErr(err)
	}
	return ToModel(row), nil
}
