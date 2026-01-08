package auth_tokens

import (
	"context"
	"database/sql"

	"github.com/Masterminds/squirrel"
	"github.com/google/uuid"
)

func (r *AuthTokensRepository) Logout(ctx context.Context, userID uuid.UUID) error {
	query := r.sb.Update(authTokensTable).
		Set(authTokensTableColumnAccessToken, sql.NullString{}).
		Set(authTokensTableColumnRefreshToken, sql.NullString{}).
		Where(squirrel.Eq{authTokensTableColumnUserID: userID})

	tag, err := r.pool.Execx(ctx, query)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}
