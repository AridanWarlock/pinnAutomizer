package refresh_tokens

import (
	"context"

	. "github.com/AridanWarlock/pinnAutomizer/auth/internal/adapter/postgres/schema"
	"github.com/AridanWarlock/pinnAutomizer/auth/internal/domain"
	sq "github.com/Masterminds/squirrel"
)

func (r *Repository) GetRefreshTokenByHash(ctx context.Context, hash string) (domain.RefreshToken, error) {
	q := r.sb.Select(RefreshTokensTableColumns...).
		From(RefreshTokensTable).
		Where(sq.Eq{RefreshTokensTableColumnHash: hash})

	var out RefreshTokenRaw
	if err := r.pool.Getx(ctx, &out, q); err != nil {
		return domain.RefreshToken{}, err
	}

	return ToModel(out), nil
}
