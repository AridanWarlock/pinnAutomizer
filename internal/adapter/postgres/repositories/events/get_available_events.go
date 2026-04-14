package events

import (
	"context"
	"fmt"

	. "github.com/AridanWarlock/pinnAutomizer/internal/adapter/postgres/schema"
	"github.com/AridanWarlock/pinnAutomizer/internal/domain"
	"github.com/AridanWarlock/pinnAutomizer/pkg/errs"
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
		Select(EventsTableColumns...).
		From(EventsTable).
		OrderBy(EventsTableColumnCreatedAt).
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
