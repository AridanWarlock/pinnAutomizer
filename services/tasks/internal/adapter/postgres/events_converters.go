package postgres

import (
	"time"

	"github.com/AridanWarlock/pinnAutomizer/pkg/core"
	"github.com/AridanWarlock/pinnAutomizer/tasks/internal/domain"
)

type EventRow struct {
	IdKey core.IdempotencyKey `db:"id_key"`

	Topic string `db:"topic"`
	Data  []byte `db:"data"`

	CrestedAt time.Time `db:"created_at"`
}

func (e EventRow) Values() []any {
	return []any{
		e.IdKey,
		e.Topic,
		e.Data,
		e.CrestedAt,
	}
}

func ToEventModel(r EventRow) domain.Event {
	return domain.Event{
		IdKey:     r.IdKey,
		Topic:     r.Topic,
		Data:      r.Data,
		CreatedAt: r.CrestedAt,
	}
}

func FromEventModel(e domain.Event) EventRow {
	return EventRow{
		IdKey:     e.IdKey,
		Topic:     e.Topic,
		Data:      e.Data,
		CrestedAt: e.CreatedAt,
	}
}
