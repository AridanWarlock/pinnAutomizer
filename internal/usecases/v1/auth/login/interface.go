package authLogin

import (
	"context"
	"time"

	"github.com/AridanWarlock/pinnAutomizer/internal/domain"
	"github.com/AridanWarlock/pinnAutomizer/pkg/validate"
)

type Input struct {
	Login    string `validate:"required,min=5,alphanum"`
	Password string `validate:"required,min=5"`

	Fingerprint domain.Fingerprint
}

func (i Input) Validate() error {
	if err := validate.V.Struct(i); err != nil {
		return err
	}

	return i.Fingerprint.Validate()
}

type Output struct {
	AccessToken           domain.AccessToken
	RefreshTokenString    string
	RefreshTokenExpiresAt time.Time
}

type Usecase interface {
	Login(ctx context.Context, in Input) (Output, error)
}
