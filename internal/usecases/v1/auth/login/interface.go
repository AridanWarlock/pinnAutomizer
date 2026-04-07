package authLogin

import (
	"context"
	"time"

	"github.com/AridanWarlock/pinnAutomizer/pkg/validate"
)

type Input struct {
	Login    string `validate:"required,min=5,alphanum"`
	Password string `validate:"required,min=5"`

	Fingerprint []byte `validate:"required,len=32"`
}

func (i Input) Validate() error {
	return validate.V.Struct(i)
}

type Output struct {
	AccessTokenString     string
	RefreshTokenString    string
	RefreshTokenExpiresAt time.Time
}

type Usecase interface {
	Login(ctx context.Context, in Input) (Output, error)
}
