package events

import (
	"context"
	"strings"

	. "github.com/AridanWarlock/pinnAutomizer/pinn/internal/adapter/postgres/schema"
	"github.com/AridanWarlock/pinnAutomizer/pinn/internal/domain"
)

func (r *Repository) PublishEvent(ctx context.Context, event domain.Event) (domain.Event, error) {
	row := FromModel(event)

	query := r.sb.
		Insert(EventsTable).
		Columns(EventsTableColumns...).
		Values(row.Values()...).
		Suffix("RETURNING " + strings.Join(EventsTableColumns, ","))

	var outRow EventRow
	if err := r.pool.Getx(ctx, &outRow, query); err != nil {
		return domain.Event{}, err
	}

	return ToModel(outRow), nil
}
