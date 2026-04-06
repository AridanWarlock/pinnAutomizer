package auth_register

import (
	"github.com/AridanWarlock/pinnAutomizer/internal/domain"
	"github.com/AridanWarlock/pinnAutomizer/pkg/validate"
)

type Input struct {
	Login             string `validate:"required,min=5,alphanum"`
	Password          string `validate:"required,min=5"`
	PasswordConfirmed string `validate:"required,eqfield=Password"`
}

func (i Input) Validate() error {
	return validate.V.Struct(i)
}

type Output struct {
	User domain.User `json:"user"`
}
