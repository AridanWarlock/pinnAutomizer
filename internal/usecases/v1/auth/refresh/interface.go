package authRefresh

import (
	"context"
	"time"

	"github.com/AridanWarlock/pinnAutomizer/internal/domain"
	"github.com/AridanWarlock/pinnAutomizer/pkg/validate"
)

type Input struct {
	RefreshTokenString string `validate:"required,base64rawurl,len=43"`
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
	Refresh(ctx context.Context, in Input) (Output, error)
}
