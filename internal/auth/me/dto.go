package me

import (
	"pinnAutomizer/pkg/validate"

	"github.com/google/uuid"
)

type Input struct {
	ID uuid.UUID `validate:"required,uuid"`
}

func (i Input) Validate() error {
	return validate.V.Struct(i)
}

type Output struct {
	ID    uuid.UUID
	Login string
}
