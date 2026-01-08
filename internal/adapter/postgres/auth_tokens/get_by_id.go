package auth_tokens

import (
	"context"
	"errors"
	"pinnAutomizer/internal/domain"

	"github.com/Masterminds/squirrel"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

func (r *AuthTokensRepository) GetAuthTokensByID(ctx context.Context, userId uuid.UUID) (*domain.AuthToken, error) {
	query := r.sb.
		Select(authTokensColumns...).
		From(authTokensTable).
		Where(squirrel.Eq{authTokensTableColumnUserID: userId})

	var row AuthTokenRow
	if err := r.pool.Getx(ctx, &row, query); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, err
	}
	return ToModel(&row), nil
}
