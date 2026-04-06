package user_sessions

import (
	"context"
	"fmt"

	"github.com/AridanWarlock/pinnAutomizer/internal/adapter/postgres/pgerr"
	. "github.com/AridanWarlock/pinnAutomizer/internal/adapter/postgres/schema"
	"github.com/AridanWarlock/pinnAutomizer/internal/domain"
	"github.com/AridanWarlock/pinnAutomizer/internal/errs"

	sq "github.com/Masterminds/squirrel"
	"github.com/google/uuid"
)

func (r *Repository) GetUserSessionById(ctx context.Context, id uuid.UUID) (domain.UserSession, error) {
	q := r.sb.Select(UserSessionsTableColumns...).
		From(UserSessionsTable).
		Where(sq.Eq{UserSessionsTableColumnID: id})

	var row UserSessionRaw
	if err := r.pool.Getx(ctx, &row, q); err != nil {
		if pgerr.IsNotFound(err) {
			return domain.UserSession{}, fmt.Errorf(
				"session with id=%v: %w",
				id,
				errs.ErrNotFound,
			)
		}
		return domain.UserSession{}, pgerr.ScanErr(err)
	}

	return ToModel(row), nil
}
