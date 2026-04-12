package authLogin

import (
	"context"
	"time"

	"github.com/AridanWarlock/pinnAutomizer/internal/domain"
	"github.com/AridanWarlock/pinnAutomizer/pkg/validate"
)

type Input struct {
	Login    string `validate:"required"`
	Password string `validate:"required,min=5"`
}

func (i Input) Validate() error {
	return validate.V.Struct(i)
}

type Output struct {
	AccessToken           domain.AccessToken
	RefreshTokenString    string
	RefreshTokenExpiresAt time.Time
}

type Usecase interface {
	Login(ctx context.Context, in Input) (Output, error)
}
