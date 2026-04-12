package refresh_tokens

import (
	"context"

	"github.com/AridanWarlock/pinnAutomizer/internal/adapter/postgres/pgerr"
	. "github.com/AridanWarlock/pinnAutomizer/internal/adapter/postgres/schema"
	"github.com/AridanWarlock/pinnAutomizer/internal/domain"
	"github.com/AridanWarlock/pinnAutomizer/internal/errs"
	sq "github.com/Masterminds/squirrel"
)

func (r *Repository) RotateRefreshToken(
	ctx context.Context,
	oldHash string,
	newHash string,
	newJti domain.Jti,
) error {
	q := r.sb.Update(RefreshTokensTable).
		Set(RefreshTokensTableColumnHash, newHash).
		Set(RefreshTokensTableColumnJti, newJti).
		Where(sq.Eq{RefreshTokensTableColumnHash: oldHash})

	tag, err := r.pool.Execx(ctx, q)
	if err != nil {
		return pgerr.ScanErr(err)
	}
	if tag.RowsAffected() != 1 {
		return errs.ErrNotFound
	}
	return nil
}
