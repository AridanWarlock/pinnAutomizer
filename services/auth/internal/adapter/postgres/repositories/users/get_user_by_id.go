package users

import (
	"context"

	. "github.com/AridanWarlock/pinnAutomizer/auth/internal/adapter/postgres/schema"
	"github.com/AridanWarlock/pinnAutomizer/auth/internal/domain"
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
		return domain.User{}, err
	}
	return ToModel(row), nil
}
