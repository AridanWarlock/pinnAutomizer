package user_sessions

import (
	"context"

	"github.com/AridanWarlock/pinnAutomizer/internal/adapter/postgres/pgerr"
	. "github.com/AridanWarlock/pinnAutomizer/internal/adapter/postgres/schema"
	"github.com/AridanWarlock/pinnAutomizer/internal/domain"
	"github.com/AridanWarlock/pinnAutomizer/internal/errs"

	sq "github.com/Masterminds/squirrel"
	"github.com/google/uuid"
)

func (r *Repository) Logout(ctx context.Context, userID uuid.UUID, fingerprint domain.Fingerprint) error {
	q := r.sb.
		Delete(UserSessionsTable).
		Where(sq.Eq{UserSessionsTableColumnUserID: userID, UserSessionsTableColumnFingerprint: fingerprint})

	tag, err := r.pool.Execx(ctx, q)
	if err != nil {
		return pgerr.ScanErr(err)
	}
	if tag.RowsAffected() != 1 {
		return errs.ErrNotFound
	}
	return nil
}
