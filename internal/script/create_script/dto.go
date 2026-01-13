package create_script

import (
	"github.com/google/uuid"
	"pinnAutomizer/pkg/validate"
)

type Input struct {
	Filename string    `validate:"required"`
	Path     string    `validate:"required"`
	UserID   uuid.UUID `validate:"required,uuid"`
}

func (i Input) Validate() error {
	return validate.V.Struct(i)
}

type Output struct {
	ID       uuid.UUID
	Filename string
	Text     string
}
