package create_user

import (
	"context"
	"pinnAutomizer/internal/adapter/postgres/pg_errors"
	"pinnAutomizer/internal/adapter/postgres/schema"
	"pinnAutomizer/internal/domain"
	"strings"
)

func (r *Repository) CreateUser(ctx context.Context, in *domain.User) (*domain.User, error) {
	userRow := FromModel(in)
	queryUserInsert := r.sb.
		Insert(schema.UsersTable).
		Columns(schema.UsersTableColumns...).
		Values(userRow.Values()...).
		Suffix("RETURNING " + strings.Join(schema.UsersTableColumns, ","))

	authTokenRow := &AuthTokenRow{
		UserID: userRow.ID,
	}

	queryAuthTokenInsert := r.sb.
		Insert(schema.AuthTokensTable).
		Columns(schema.AuthTokensColumns...).
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
			return pg_errors.ErrNotFound
		}

		return nil
	})

	if err != nil {
		return nil, err
	}
	return outRow.ToModel(), nil
}
