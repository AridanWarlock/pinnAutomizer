package authRefresh

import (
	"context"
	"time"

	"github.com/AridanWarlock/pinnAutomizer/pkg/core"
	"github.com/AridanWarlock/pinnAutomizer/pkg/validate"
)

type Input struct {
	RefreshTokenString string `validate:"required,base64rawurl,len=43"`
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
	Refresh(ctx context.Context, in Input) (Output, error)
}
