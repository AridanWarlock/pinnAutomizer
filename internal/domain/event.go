package domain

import (
	"time"

	"github.com/google/uuid"
)

type Event struct {
	ID uuid.UUID

	Topic string
	Data  []byte

	CreatedAt time.Time
}

func NewEvent(topic string, data []byte) Event {
	return Event{
		ID: uuid.New(),

		Topic: topic,
		Data:  data,

		CreatedAt: time.Now(),
	}
}
