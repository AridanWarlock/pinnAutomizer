package domain

import (
	"pinnAutomizer/pkg/validate"
	"time"

	"github.com/google/uuid"
)

type Event struct {
	ID uuid.UUID `validate:"required" json:"id"`

	Topic string `validate:"required" json:"topic"`
	Data  []byte `validate:"required" json:"data"`

	CreatedAt time.Time `validate:"required" json:"created_at"`
}

func (e Event) Validate() error {
	return validate.V.Struct(e)
}

func NewEvent(topic string, data []byte) Event {
	return Event{
		ID: uuid.New(),

		Topic: topic,
		Data:  data,

		CreatedAt: time.Now(),
	}
}
