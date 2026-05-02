package domain

import (
	"fmt"
	"time"

	"github.com/AridanWarlock/pinnAutomizer/pkg/core"
	"github.com/AridanWarlock/pinnAutomizer/pkg/errs"
	"github.com/AridanWarlock/pinnAutomizer/pkg/validate"
)

type Event struct {
	IdKey core.IdempotencyKey

	Topic string `validate:"required" json:"topic"`
	Data  []byte `validate:"required" json:"data"`

	CreatedAt time.Time `validate:"required" json:"created_at"`
}

func NewEvent(idKey core.IdempotencyKey, topic string, data []byte) (Event, error) {
	e := Event{
		IdKey: idKey,

		Topic: topic,
		Data:  data,

		CreatedAt: time.Now(),
	}

	if err := e.Validate(); err != nil {
		return Event{}, fmt.Errorf("%w: %v", errs.ErrInvalidArgument, err)
	}

	return e, nil
}

func (e Event) Validate() error {
	return validate.V.Struct(e)
}
