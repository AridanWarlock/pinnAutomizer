package events

import (
	"context"

	"github.com/AridanWarlock/pinnAutomizer/internal/adapter/postgres/pgerr"
	. "github.com/AridanWarlock/pinnAutomizer/internal/adapter/postgres/schema"
	"github.com/AridanWarlock/pinnAutomizer/pkg/logger"

	. "github.com/Masterminds/squirrel"
	"github.com/google/uuid"
)

func (r *Repository) DeleteEventsByIDs(ctx context.Context, ids []uuid.UUID) error {
	log := logger.FromContext(ctx)

	query := r.sb.
		Delete(EventsTable).
		Where(Eq{EquationsTableColumnID: ids})

	tag, err := r.pool.Execx(ctx, query)
	if err != nil {
		return pgerr.ScanErr(err)
	}

	expected := int64(len(ids))
	actual := tag.RowsAffected()

	if actual < expected {
		log.Info().Int64("deleted_count", actual).
			Int64("requested_count", expected).
			Interface("ids", ids).
			Msg("delete events by id")
	}

	return nil
}
