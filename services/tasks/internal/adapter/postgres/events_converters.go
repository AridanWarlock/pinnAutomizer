package postgres

import (
	"time"

	"github.com/AridanWarlock/pinnAutomizer/tasks/internal/domain"
	"github.com/google/uuid"
)

type EventRow struct {
	ID uuid.UUID `db:"id"`

	Topic string `db:"topic"`
	Data  []byte `db:"data"`

	CrestedAt time.Time `db:"created_at"`
}

func (e EventRow) Values() []any {
	return []any{
		e.ID,
		e.Topic,
		e.Data,
		e.CrestedAt,
	}
}

func ToEventModel(r EventRow) domain.Event {
	return domain.Event{
		ID:        r.ID,
		Topic:     r.Topic,
		Data:      r.Data,
		CreatedAt: r.CrestedAt,
	}
}

func FromEventModel(e domain.Event) EventRow {
	return EventRow{
		ID:        e.ID,
		Topic:     e.Topic,
		Data:      e.Data,
		CrestedAt: e.CreatedAt,
	}
}
