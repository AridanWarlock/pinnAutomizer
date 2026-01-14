package refresh

import (
	"pinnAutomizer/internal/domain"
	"pinnAutomizer/pkg/validate"
)

type Input struct {
	RefreshToken string `validate:"required"`
}

func (i Input) Validate() error {
	return validate.V.Struct(i)
}

type Output struct {
	AccessToken domain.Token
}
