package auth_me

import (
	"github.com/AridanWarlock/pinnAutomizer/pkg/validate"

	"github.com/google/uuid"
)

type Input struct {
	UserID uuid.UUID `validate:"required,uuid"`
}

func (i Input) Validate() error {
	return validate.V.Struct(i)
}

type Output struct {
	UserID uuid.UUID
	Login  string
}
