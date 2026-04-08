package authLogout

import (
	"context"
	"fmt"

	"github.com/AridanWarlock/pinnAutomizer/internal/domain"
	"github.com/AridanWarlock/pinnAutomizer/pkg/validate"

	"github.com/google/uuid"
)

type Input struct {
	UserID      uuid.UUID `validate:"required,uuid"`
	Fingerprint domain.Fingerprint
}

func (i Input) Validate() error {
	if err := i.Fingerprint.Validate(); err != nil {
		return fmt.Errorf("validate fingerprint: %w", err)
	}

	return validate.V.Struct(i)
}

type Usecase interface {
	Logout(ctx context.Context, in Input) error
}
