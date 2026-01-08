package create_user

import (
	"context"
	"pinnAutomizer/internal/domain"
	"strings"
)

func (r *CreateUserRepository) CreateUser(ctx context.Context, in *domain.User) (*domain.User, error) {
	userRow := FromModel(in)
	queryUserInsert := r.sb.
		Insert(usersTable).
		Columns(usersTableColumns...).
		Values(userRow.Values()...).
		Suffix("RETURNING " + strings.Join(usersTableColumns, ","))

	authTokenRow := &AuthTokenRow{
		UserID: userRow.ID,
	}

	queryAuthTokenInsert := r.sb.
		Insert(authTokensTable).
		Columns(authTokensColumns...).
		Values(authTokenRow.Values()...)

	var outRow CreateUserRow
	err := r.pool.Wrap(ctx, func(ctx context.Context) error {
		if err := r.pool.Getx(ctx, &outRow, queryUserInsert); err != nil {
			return err
		}

		tag, err := r.pool.Execx(ctx, queryAuthTokenInsert)
		if err != nil {
			return err
		}
		if tag.RowsAffected() == 0 {
			return ErrNotFound
		}

		return nil
	})

	if err != nil {
		return nil, err
	}
	return outRow.ToModel(), nil
}
