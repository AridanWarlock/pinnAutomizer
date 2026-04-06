package auth_me

import (
	"context"
	"fmt"

	"github.com/AridanWarlock/pinnAutomizer/internal/domain"
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

func (u *usecase) Me(ctx context.Context, in Input) (Output, error) {
	if err := in.Validate(); err != nil {
		return Output{}, domain.ErrInputValidation
	}

	user, err := u.postgres.GetUserByID(ctx, in.UserID)
	if err != nil {
		return Output{}, fmt.Errorf("get user by id from postgres: %w", err)
	}

	return Output{
		UserID: user.ID,
		Login:  user.Login,
	}, nil
}
