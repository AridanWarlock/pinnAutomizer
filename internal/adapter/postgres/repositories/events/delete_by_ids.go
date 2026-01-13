package events

import (
	"context"
	"fmt"
	"pinnAutomizer/internal/adapter/postgres/schema"

	sq "github.com/Masterminds/squirrel"
	"github.com/google/uuid"
)

func (r *Repository) DeleteEventsByIDs(ctx context.Context, ids []uuid.UUID) error {
	query := r.sb.
		Delete(schema.EventsTable).
		Where(sq.Eq{schema.EquationsTableColumnID: ids})

	tag, err := r.pool.Execx(ctx, query)

	if err != nil {
		return err
	}
	if tag.RowsAffected() != int64(len(ids)) {
		return fmt.Errorf("expected %d rows affected, got %d", len(ids), tag.RowsAffected())
	}
	return nil
}
