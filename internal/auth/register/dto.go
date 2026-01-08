package register

import "github.com/go-playground/validator/v10"

type Input struct {
	Login             string `validate:"required,min=5,alphanum"`
	Password          string `validate:"required,min=5"`
	PasswordConfirmed string `validate:"required,eqfield=Password"`
}

func (i Input) Validate(validate *validator.Validate) error {
	return validate.Struct(i)
}
