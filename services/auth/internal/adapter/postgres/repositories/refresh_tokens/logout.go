package refresh_tokens

import (
	"context"

	. "github.com/AridanWarlock/pinnAutomizer/auth/internal/adapter/postgres/schema"
	"github.com/AridanWarlock/pinnAutomizer/pkg/core"
	"github.com/AridanWarlock/pinnAutomizer/pkg/errs"
	sq "github.com/Masterminds/squirrel"
	"github.com/google/uuid"
)

func (r *Repository) Logout(
	ctx context.Context,
	userID uuid.UUID,
	fingerprint core.Fingerprint,
) error {
	q := r.sb.Delete(RefreshTokensTable).
		Where(sq.Eq{
			RefreshTokensTableColumnUserID:      userID,
			RefreshTokensTableColumnFingerprint: fingerprint,
		})

	tag, err := r.pool.Execx(ctx, q)
	if err != nil {
		return err
	}
	if tag.RowsAffected() != 1 {
		return errs.ErrNotFound
	}
	return nil
}
