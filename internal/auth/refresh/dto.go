package refresh

import (
	"pinnAutomizer/internal/domain"

	"github.com/go-playground/validator/v10"
)

type Input struct {
	RefreshToken string `validate:"required"`
}

func (i Input) Validate(validate *validator.Validate) error {
	return validate.Struct(i)
}

type Output struct {
	AccessToken domain.Token
}
