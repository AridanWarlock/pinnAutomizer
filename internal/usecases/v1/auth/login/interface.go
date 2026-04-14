package authLogin

import (
	"context"
	"time"

	"github.com/AridanWarlock/pinnAutomizer/pkg/core"
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
	AccessToken           core.AccessToken
	RefreshTokenString    string
	RefreshTokenExpiresAt time.Time
}

type Usecase interface {
	Login(ctx context.Context, in Input) (Output, error)
}
