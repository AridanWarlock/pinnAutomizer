package auth_tokens

import (
	"context"
	"database/sql"
	"pinnAutomizer/internal/adapter/postgres/schema"

	"github.com/Masterminds/squirrel"
	"github.com/google/uuid"
)

func (r *Repository) Logout(ctx context.Context, userID uuid.UUID) error {
	query := r.sb.Update(schema.AuthTokensTable).
		Set(schema.AuthTokensTableColumnAccessToken, sql.NullString{}).
		Set(schema.AuthTokensTableColumnRefreshToken, sql.NullString{}).
		Where(squirrel.Eq{schema.AuthTokensTableColumnUserID: userID})

	tag, err := r.pool.Execx(ctx, query)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}
