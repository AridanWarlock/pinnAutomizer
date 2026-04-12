package refresh_tokens

import (
	"context"

	"github.com/AridanWarlock/pinnAutomizer/internal/adapter/postgres/pgerr"
	. "github.com/AridanWarlock/pinnAutomizer/internal/adapter/postgres/schema"
	"github.com/AridanWarlock/pinnAutomizer/internal/domain"
	"github.com/AridanWarlock/pinnAutomizer/internal/errs"
	sq "github.com/Masterminds/squirrel"
)

func (r *Repository) GetRefreshTokenByHash(ctx context.Context, hash string) (domain.RefreshToken, error) {
	q := r.sb.Select(RefreshTokensTableColumns...).
		From(RefreshTokensTable).
		Where(sq.Eq{RefreshTokensTableColumnHash: hash})

	var out RefreshTokenRaw
	if err := r.pool.Getx(ctx, &out, q); err != nil {
		if pgerr.IsNotFound(err) {
			return domain.RefreshToken{}, errs.ErrNotFound
		}
		return domain.RefreshToken{}, pgerr.ScanErr(err)
	}

	return ToModel(out), nil
}
