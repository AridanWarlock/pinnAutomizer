package auth_logout

import (
	"context"
	"fmt"

	"github.com/AridanWarlock/pinnAutomizer/internal/domain"
	"github.com/google/uuid"
)

type Postgres interface {
	Logout(ctx context.Context, userID uuid.UUID) error
}

type Usecase struct {
	postgres Postgres
}

func New(
	postgres Postgres,
) *Usecase {
	return &Usecase{
		postgres: postgres,
	}
}

func (u *Usecase) Logout(ctx context.Context, in Input) error {
	if err := in.Validate(); err != nil {
		return domain.ErrInputValidation
	}

	if err := u.postgres.Logout(ctx, in.ID); err != nil {
		return fmt.Errorf("delete session from postgres: %w", err)
	}
	return nil
}
