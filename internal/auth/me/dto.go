package me

import (
	"github.com/google/uuid"
	"pinnAutomizer/pkg/validate"
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
