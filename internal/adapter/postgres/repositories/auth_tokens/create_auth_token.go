package auth_tokens

import (
	"context"
	. "pinnAutomizer/internal/adapter/postgres/schema"
	"pinnAutomizer/internal/domain"
	"strings"

	"github.com/google/uuid"
)

func (r *Repository) CreateAuthToken(ctx context.Context, userID uuid.UUID) (domain.AuthToken, error) {
	authTokenRow := AuthTokenRow{
		UserID: userID,
	}

	query := r.sb.
		Insert(AuthTokensTable).
		Columns(AuthTokensColumns...).
		Values(authTokenRow.Values()...).
		Suffix("RETURNING " + strings.Join(AuthTokensColumns, ","))

	var outRow AuthTokenRow
	if err := r.pool.Getx(ctx, &outRow, query); err != nil {
		return domain.AuthToken{}, err
	}
	return ToModel(outRow), nil
}
