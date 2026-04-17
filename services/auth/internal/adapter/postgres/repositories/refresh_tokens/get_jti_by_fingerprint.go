package refresh_tokens

import (
	"context"

	. "github.com/AridanWarlock/pinnAutomizer/auth/internal/adapter/postgres/schema"
	"github.com/AridanWarlock/pinnAutomizer/pkg/core"
	sq "github.com/Masterminds/squirrel"
	"github.com/google/uuid"
)

func (r *Repository) GetJtiByFingerprint(
	ctx context.Context,
	userID uuid.UUID,
	fingerprint core.Fingerprint,
) (core.Jti, error) {
	q := r.sb.Select(RefreshTokensTableColumnJti).
		From(RefreshTokensTable).
		Where(sq.Eq{
			RefreshTokensTableColumnUserID:      userID,
			RefreshTokensTableColumnFingerprint: fingerprint,
		})

	var jti core.Jti
	if err := r.pool.Getx(ctx, &jti, q); err != nil {
		return core.Jti{}, err
	}
	return jti, nil
}
