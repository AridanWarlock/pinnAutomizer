package login

import (
	"pinnAutomizer/internal/domain"
	"pinnAutomizer/pkg/validate"
)

type Input struct {
	Login    string `validate:"required,min=5,alphanum"`
	Password string `validate:"required,min=5"`
}

func (i Input) Validate() error {
	return validate.V.Struct(i)
}

type Output struct {
	AccessToken  domain.Token
	RefreshToken domain.Token
}
