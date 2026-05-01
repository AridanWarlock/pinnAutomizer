package postgres

import (
	"context"
	"strings"

	"github.com/AridanWarlock/pinnAutomizer/auth/internal/domain"
	sq "github.com/Masterminds/squirrel"
	"github.com/google/uuid"
)

func (r *Repository) CreateUser(ctx context.Context, user domain.User) (domain.User, error) {
	row := fromUserModel(user)
	query := r.sb.
		Insert(UsersTable).
		Columns(UsersColumns...).
		Values(row.Values()...).
		Suffix("RETURNING " + strings.Join(UsersColumns, ","))

	var outRow UserRow
	if err := r.pool.Getx(ctx, &outRow, query); err != nil {
		return domain.User{}, err
	}
	return toUserModel(outRow), nil
}

func (r *Repository) GetUserByID(ctx context.Context, id uuid.UUID) (domain.User, error) {
	query := r.sb.
		Select(UsersColumns...).
		From(UsersTable).
		Where(sq.Eq{UsersID: id})

	var row UserRow
	if err := r.pool.Getx(ctx, &row, query); err != nil {
		return domain.User{}, err
	}
	return toUserModel(row), nil
}

func (r *Repository) GetUserByLogin(ctx context.Context, login string) (domain.User, error) {
	query := r.sb.
		Select(UsersColumns...).
		From(UsersTable).
		Where(sq.Eq{UsersLogin: login})

	var outRow UserRow
	if err := r.pool.Getx(ctx, &outRow, query); err != nil {
		return domain.User{}, err
	}

	return toUserModel(outRow), nil
}
