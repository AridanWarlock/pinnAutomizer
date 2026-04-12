package refresh_tokens

import (
	"context"

	"github.com/AridanWarlock/pinnAutomizer/internal/adapter/postgres/pgerr"
	. "github.com/AridanWarlock/pinnAutomizer/internal/adapter/postgres/schema"
	"github.com/AridanWarlock/pinnAutomizer/internal/domain"
	"github.com/AridanWarlock/pinnAutomizer/internal/errs"
	sq "github.com/Masterminds/squirrel"
	"github.com/google/uuid"
)

func (r *Repository) GetJtiByFingerprint(
	ctx context.Context,
	userID uuid.UUID,
	fingerprint domain.Fingerprint,
) (domain.Jti, error) {
	q := r.sb.Select(RefreshTokensTableColumnJti).
		From(RefreshTokensTable).
		Where(sq.Eq{
			RefreshTokensTableColumnUserID:      userID,
			RefreshTokensTableColumnFingerprint: fingerprint,
		})

	var jti domain.Jti
	if err := r.pool.Getx(ctx, &jti, q); err != nil {
		if pgerr.IsNotFound(err) {
			return domain.Jti{}, errs.ErrNotFound
		}
		return domain.Jti{}, pgerr.ScanErr(err)
	}
	return jti, nil
}
