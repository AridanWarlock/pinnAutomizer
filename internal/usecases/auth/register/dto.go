package register

import (
	"pinnAutomizer/pkg/validate"
)

type Input struct {
	Login             string `validate:"required,min=5,alphanum"`
	Password          string `validate:"required,min=5"`
	PasswordConfirmed string `validate:"required,eqfield=Password"`
}

func (i Input) Validate() error {
	return validate.V.Struct(i)
}
