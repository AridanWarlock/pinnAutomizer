package user_sessions

import (
	"context"
	"fmt"

	"github.com/AridanWarlock/pinnAutomizer/internal/adapter/postgres/pgerr"
	. "github.com/AridanWarlock/pinnAutomizer/internal/adapter/postgres/schema"
	"github.com/AridanWarlock/pinnAutomizer/internal/errs"

	sq "github.com/Masterminds/squirrel"
	"github.com/google/uuid"
)

func (r *Repository) Logout(ctx context.Context, sessionId uuid.UUID) error {
	q := r.sb.
		Delete(UserSessionsTable).
		Where(sq.Eq{UserSessionsTableColumnID: sessionId})

	tag, err := r.pool.Execx(ctx, q)
	if err != nil {
		return pgerr.ScanErr(err)
	}
	if tag.RowsAffected() != 1 {
		return fmt.Errorf(
			"session with id=%v: %w",
			sessionId,
			errs.ErrNotFound,
		)
	}
	return nil
}
