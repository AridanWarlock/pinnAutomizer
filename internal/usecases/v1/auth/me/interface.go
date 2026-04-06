package auth_me

import (
	"context"

	"github.com/AridanWarlock/pinnAutomizer/pkg/validate"

	"github.com/google/uuid"
)

type Input struct {
	UserID uuid.UUID `validate:"required,uuid"`
}

func (i Input) Validate() error {
	return validate.V.Struct(i)
}

type Output struct {
	UserID uuid.UUID
	Login  string
}

type Usecase interface {
	Me(ctx context.Context, in Input) (Output, error)
}
