package user_sessions

import (
	"context"
	"errors"

	"github.com/AridanWarlock/pinnAutomizer/internal/adapter/postgres/pg_errors"
	. "github.com/AridanWarlock/pinnAutomizer/internal/adapter/postgres/schema"
	"github.com/AridanWarlock/pinnAutomizer/internal/domain"

	sq "github.com/Masterminds/squirrel"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

func (r *Repository) GetUserSessionById(ctx context.Context, id uuid.UUID) (domain.UserSession, error) {
	q := r.sb.Select(UserSessionsTableColumns...).
		From(UserSessionsTable).
		Where(sq.Eq{UserSessionsTableColumnID: id})

	var row UserSessionRaw
	if err := r.pool.Getx(ctx, &row, q); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return domain.UserSession{}, pg_errors.ErrNotFound
		}
		return domain.UserSession{}, err
	}

	return ToModel(row), nil
}
