package authRefresh

import (
	"context"

	"github.com/AridanWarlock/pinnAutomizer/internal/domain"
	"github.com/AridanWarlock/pinnAutomizer/pkg/validate"
)

type Input struct {
	RefreshTokenString string `validate:"required"`
	Fingerprint        domain.Fingerprint
}

func (i Input) Validate() error {
	if err := validate.V.Struct(i); err != nil {
		return err
	}

	return i.Fingerprint.Validate()
}

type Output struct {
	AccessToken domain.AccessToken
}

type Usecase interface {
	Refresh(ctx context.Context, in Input) (Output, error)
}
