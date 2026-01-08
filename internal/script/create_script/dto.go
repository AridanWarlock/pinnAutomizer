package create_script

import (
	"github.com/go-playground/validator/v10"
	"github.com/google/uuid"
)

type Input struct {
	Filename string    `validate:"required"`
	Path     string    `validate:"required"`
	UserID   uuid.UUID `validate:"required,uuid"`
}

func (i Input) Validate(validate *validator.Validate) error {
	return validate.Struct(i)
}

type Output struct {
	ID       uuid.UUID
	Filename string
	Text     string
}
