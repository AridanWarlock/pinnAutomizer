package auth_tokens

import (
	"context"
	"pinnAutomizer/internal/domain"

	"github.com/Masterminds/squirrel"
)

func (r *AuthTokensRepository) Login(ctx context.Context, authToken *domain.AuthToken) error {
	query := r.sb.Update(authTokensTable).
		Set(authTokensTableColumnAccessToken, authToken.AccessToken).
		Set(authTokensTableColumnRefreshToken, authToken.RefreshToken).
		Where(squirrel.Eq{authTokensTableColumnUserID: authToken.UserID})

	tag, err := r.pool.Execx(ctx, query)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}
