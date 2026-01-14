package auth_tokens

import (
	"context"
	"errors"
	. "pinnAutomizer/internal/adapter/postgres/pg_errors"
	. "pinnAutomizer/internal/adapter/postgres/schema"
	"pinnAutomizer/internal/domain"

	. "github.com/Masterminds/squirrel"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

func (r *Repository) GetAuthTokensByID(ctx context.Context, userId uuid.UUID) (domain.AuthToken, error) {
	query := r.sb.
		Select(AuthTokensColumns...).
		From(AuthTokensTable).
		Where(Eq{AuthTokensTableColumnUserID: userId})

	var row AuthTokenRow
	if err := r.pool.Getx(ctx, &row, query); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return domain.AuthToken{}, ErrNotFound
		}
		return domain.AuthToken{}, err
	}
	return ToModel(row), nil
}
