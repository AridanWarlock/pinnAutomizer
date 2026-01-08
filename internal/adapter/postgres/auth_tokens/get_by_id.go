package auth_tokens

import (
	"context"
	"errors"
	"pinnAutomizer/internal/adapter/postgres/schema"
	"pinnAutomizer/internal/domain"

	"github.com/Masterminds/squirrel"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

func (r *Repository) GetAuthTokensByID(ctx context.Context, userId uuid.UUID) (*domain.AuthToken, error) {
	query := r.sb.
		Select(schema.AuthTokensColumns...).
		From(schema.AuthTokensTable).
		Where(squirrel.Eq{schema.AuthTokensTableColumnUserID: userId})

	var row AuthTokenRow
	if err := r.pool.Getx(ctx, &row, query); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, err
	}
	return ToModel(&row), nil
}
