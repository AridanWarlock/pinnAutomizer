package domain

import (
	"time"

	"github.com/AridanWarlock/pinnAutomizer/pkg/validate"

	"github.com/google/uuid"
)

type IdempotencyKey struct {
	Key       string    `validate:"required" json:"key"`
	UserID    uuid.UUID `validate:"required,uuid" json:"user_id"`
	Data      []byte    `validate:"required" json:"data"`
	Error     string    `json:"error"`
	CreatedAt time.Time `validate:"required" json:"created_at"`
}

func NewIdempotencyKey(key string, data []byte, err error) IdempotencyKey {
	var errString string
	if err != nil {
		errString = err.Error()
	}

	return IdempotencyKey{
		Key:       key,
		Data:      data,
		Error:     errString,
		CreatedAt: time.Now(),
	}
}

func (i IdempotencyKey) Validate() error {
	return validate.V.Struct(i)
}
