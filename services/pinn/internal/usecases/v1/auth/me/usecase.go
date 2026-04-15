package authMe

import (
	"context"
	"fmt"

	"github.com/AridanWarlock/pinnAutomizer/pinn/internal/domain"
	"github.com/AridanWarlock/pinnAutomizer/pkg/core"
	"github.com/google/uuid"
)

type Postgres interface {
	GetUserByID(ctx context.Context, id uuid.UUID) (domain.User, error)
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

func (u *usecase) Me(ctx context.Context) (Output, error) {
	auth := core.MustAuthInfoFromContext(ctx)

	user, err := u.postgres.GetUserByID(ctx, auth.UserID)
	if err != nil {
		return Output{}, fmt.Errorf("get user by id from postgres: %w", err)
	}

	return Output{
		User: user,
	}, nil
}
