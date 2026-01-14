package auth_tokens

import (
	"context"
	"database/sql"
	. "pinnAutomizer/internal/adapter/postgres/pg_errors"
	. "pinnAutomizer/internal/adapter/postgres/schema"

	. "github.com/Masterminds/squirrel"
	"github.com/google/uuid"
)

func (r *Repository) Logout(ctx context.Context, userID uuid.UUID) error {
	query := r.sb.Update(AuthTokensTable).
		Set(AuthTokensTableColumnAccessToken, sql.NullString{}).
		Set(AuthTokensTableColumnRefreshToken, sql.NullString{}).
		Where(Eq{AuthTokensTableColumnUserID: userID})

	tag, err := r.pool.Execx(ctx, query)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}
