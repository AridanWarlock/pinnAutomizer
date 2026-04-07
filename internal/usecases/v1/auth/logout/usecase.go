package authLogout

import (
	"context"
	"fmt"

	"github.com/AridanWarlock/pinnAutomizer/internal/errs"
	"github.com/google/uuid"
)

type Postgres interface {
	Logout(ctx context.Context, userID uuid.UUID) error
}

type usecase struct {
	postgres Postgres
}

func New(
	postgres Postgres,
) Usecase {
	return &usecase{
		postgres: postgres,
	}
}

func (u *usecase) Logout(ctx context.Context, in Input) error {
	if err := in.Validate(); err != nil {
		return fmt.Errorf("%w: %v", errs.ErrInvalidArgument, err)
	}

	if err := u.postgres.Logout(ctx, in.ID); err != nil {
		return fmt.Errorf("delete session from postgres: %w", err)
	}
	return nil
}
