package logout

import (
	"github.com/go-playground/validator/v10"
	"github.com/google/uuid"
)

type Input struct {
	ID uuid.UUID `validate:"required,uuid"`
}

func (i Input) Validate(validate *validator.Validate) error {
	return validate.Struct(i)
}
