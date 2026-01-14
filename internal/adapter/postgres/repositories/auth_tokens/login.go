package auth_tokens

import (
	"context"
	. "pinnAutomizer/internal/adapter/postgres/pg_errors"
	. "pinnAutomizer/internal/adapter/postgres/schema"
	"pinnAutomizer/internal/domain"

	. "github.com/Masterminds/squirrel"
)

func (r *Repository) Login(ctx context.Context, authToken domain.AuthToken) error {
	query := r.sb.Update(AuthTokensTable).
		Set(AuthTokensTableColumnAccessToken, authToken.AccessToken).
		Set(AuthTokensTableColumnRefreshToken, authToken.RefreshToken).
		Where(Eq{AuthTokensTableColumnUserID: authToken.UserID})

	tag, err := r.pool.Execx(ctx, query)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}
