package events

import (
	"context"
	"pinnAutomizer/internal/adapter/postgres/pg_errors"
	"pinnAutomizer/internal/adapter/postgres/schema"
	"pinnAutomizer/internal/domain"
)

func (r *Repository) GetAvailableEvents(ctx context.Context, batchSize int) ([]domain.Event, error) {
	if batchSize < 0 || batchSize > 1000 {
		return nil, pg_errors.ErrInvalidBatchSize
	}

	query := r.sb.
		Select(schema.EventsTableColumns...).
		From(schema.EventsTable).
		OrderBy(schema.EventsTableColumnCreatedAt).
		Limit(uint64(batchSize)).
		Suffix("FOR UPDATE SKIP LOCKED")

	var rows []EventRow
	if err := r.pool.Selectx(ctx, &rows, query); err != nil {
		return nil, err
	}

	events := make([]domain.Event, len(rows))
	for i, row := range rows {
		events[i] = ToModel(row)
	}
	return events, nil
}
