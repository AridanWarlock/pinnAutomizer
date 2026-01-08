package login

import (
	"pinnAutomizer/internal/domain"

	"github.com/go-playground/validator/v10"
)

type Input struct {
	Login    string `validate:"required,min=5,alphanum"`
	Password string `validate:"required,min=5"`
}

func (i Input) Validate(validate *validator.Validate) error {
	return validate.Struct(i)
}

type Output struct {
	AccessToken  domain.Token
	RefreshToken domain.Token
}
