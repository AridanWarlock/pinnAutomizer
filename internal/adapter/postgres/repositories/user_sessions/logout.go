package user_sessions

import (
	"context"

	"github.com/AridanWarlock/pinnAutomizer/internal/adapter/postgres/pg_errors"
	. "github.com/AridanWarlock/pinnAutomizer/internal/adapter/postgres/schema"

	sq "github.com/Masterminds/squirrel"
	"github.com/google/uuid"
)

func (r *Repository) Logout(ctx context.Context, sessionId uuid.UUID) error {
	q := r.sb.
		Delete(UserSessionsTable).
		Where(sq.Eq{UserSessionsTableColumnID: sessionId})

	tag, err := r.pool.Execx(ctx, q)
	if err != nil {
		return err
	}
	if tag.RowsAffected() != 1 {
		return pg_errors.ErrDeleteRowsAffectedCount
	}
	return nil
}
