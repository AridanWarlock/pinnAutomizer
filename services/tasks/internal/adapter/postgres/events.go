package postgres

import (
	"context"
	"fmt"
	"strings"

	"github.com/AridanWarlock/pinnAutomizer/pkg/errs"
	"github.com/AridanWarlock/pinnAutomizer/pkg/logger"
	"github.com/AridanWarlock/pinnAutomizer/tasks/internal/domain"
	sq "github.com/Masterminds/squirrel"
	"github.com/google/uuid"
)

func (r *Repository) GetAvailableEvents(ctx context.Context, batchSize int) ([]domain.Event, error) {
	if batchSize < 0 || batchSize > 100 {
		return nil, fmt.Errorf(
			"%w: invalid batch size=%d",
			errs.ErrInvalidArgument,
			batchSize,
		)
	}

	query := r.sb.
		Select(EventsColumns...).
		From(EventsTable).
		OrderBy(EventsCreatedAt).
		Limit(uint64(batchSize)).
		Suffix("FOR UPDATE SKIP LOCKED")

	var rows []EventRow
	if err := r.pool.Selectx(ctx, &rows, query); err != nil {
		return nil, err
	}

	events := make([]domain.Event, len(rows))
	for i, row := range rows {
		events[i] = ToEventModel(row)
	}
	return events, nil
}

func (r *Repository) PublishEvent(ctx context.Context, event domain.Event) (domain.Event, error) {
	row := FromEventModel(event)

	query := r.sb.
		Insert(EventsTable).
		Columns(EventsColumns...).
		Values(row.Values()...).
		Suffix("RETURNING " + strings.Join(EventsColumns, ","))

	var outRow EventRow
	if err := r.pool.Getx(ctx, &outRow, query); err != nil {
		return domain.Event{}, err
	}

	return ToEventModel(outRow), nil
}

func (r *Repository) DeleteEventsByIDs(ctx context.Context, ids []uuid.UUID) error {
	log := logger.FromContext(ctx)

	query := r.sb.
		Delete(EventsTable).
		Where(sq.Eq{EquationsID: ids})

	tag, err := r.pool.Execx(ctx, query)
	if err != nil {
		return err
	}

	expected := len(ids)
	actual := tag.RowsAffected()

	if actual < expected {
		log.Info().Int("deleted_count", actual).
			Int("requested_count", expected).
			Interface("ids", ids).
			Msg("delete events by id")
	}

	return nil
}
