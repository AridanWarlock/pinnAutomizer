package auth_refresh

import (
	"context"

	"github.com/AridanWarlock/pinnAutomizer/internal/domain"
	"github.com/AridanWarlock/pinnAutomizer/pkg/validate"
)

type Input struct {
	RefreshTokenString string `validate:"required"`
}

func (i Input) Validate() error {
	return validate.V.Struct(i)
}

type Output struct {
	AccessToken domain.AccessToken
}

type Usecase interface {
	Refresh(ctx context.Context, in Input) (Output, error)
}
