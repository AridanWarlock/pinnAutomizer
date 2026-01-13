package events

import (
	"github.com/google/uuid"
	"pinnAutomizer/internal/domain"
	"time"
)

type EventRow struct {
	ID uuid.UUID `db:"id"`

	Topic string `db:"topic"`
	Data  []byte `db:"data"`

	CrestedAt time.Time `db:"created_at"`
}

func ToModel(r EventRow) domain.Event {
	return domain.Event{
		ID:        r.ID,
		Topic:     r.Topic,
		Data:      r.Data,
		CreatedAt: r.CrestedAt,
	}
}

func FromModel(e domain.Event) EventRow {
	return EventRow{
		ID:        e.ID,
		Topic:     e.Topic,
		Data:      e.Data,
		CrestedAt: e.CreatedAt,
	}
}
