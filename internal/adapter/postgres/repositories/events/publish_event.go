package events

import (
	"context"

	. "github.com/AridanWarlock/pinnAutomizer/internal/adapter/postgres/schema"
	"github.com/AridanWarlock/pinnAutomizer/internal/domain"
)

func (r *Repository) PublishEvent(ctx context.Context, event domain.Event) error {
	row := FromModel(event)

	query := r.sb.
		Insert(EventsTable).
		Columns(EventsTableColumns...).
		Values(row.Values()...)

	_, err := r.pool.Execx(ctx, query)
	if err != nil {
		return err
	}
	return nil
}
