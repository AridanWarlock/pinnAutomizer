package auth_tokens

import (
	"context"

	"github.com/Masterminds/squirrel"
	"github.com/google/uuid"
)

func (r *AuthTokensRepository) Refresh(ctx context.Context, userID uuid.UUID, newAccessToken string) error {
	query := r.sb.Update(authTokensTable).
		Set(authTokensTableColumnAccessToken, newAccessToken).
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
