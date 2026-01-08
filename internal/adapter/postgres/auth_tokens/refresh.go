package auth_tokens

import (
	"context"
	"pinnAutomizer/internal/adapter/postgres/schema"

	"github.com/Masterminds/squirrel"
	"github.com/google/uuid"
)

func (r *Repository) Refresh(ctx context.Context, userID uuid.UUID, newAccessToken string) error {
	query := r.sb.Update(schema.AuthTokensTable).
		Set(schema.AuthTokensTableColumnAccessToken, newAccessToken).
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
