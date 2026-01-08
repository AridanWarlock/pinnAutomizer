package me

import (
	"context"
	"pinnAutomizer/internal/domain"

	"github.com/go-playground/validator/v10"
	"github.com/google/uuid"
	"github.com/rs/zerolog"
)

type Postgres interface {
	GetUserByID(ctx context.Context, id uuid.UUID) (*domain.User, error)
}

type Usecase struct {
	postgres Postgres

	log      zerolog.Logger
	validate *validator.Validate
}

var usecase *Usecase

func New(
	postgres Postgres,
	log zerolog.Logger,
	validate *validator.Validate,
) *Usecase {
	uc := &Usecase{
		postgres: postgres,

		log:      log.With().Str("component", "usecase: auth.Me").Logger(),
		validate: validate,
	}

	usecase = uc

	return uc
}

func (u *Usecase) Me(ctx context.Context, in Input) (Output, error) {
	log := u.log.With().Ctx(ctx).Logger()

	if err := in.Validate(u.validate); err != nil {
		log.Info().
			Err(err).
			Msg("input validation error")
		return Output{}, err
	}

	user, err := u.postgres.GetUserByID(ctx, in.ID)
	if err != nil {
		log.Info().
			Err(err).
			Msg("getting user from postgres error")
		return Output{}, err
	}

	return Output{
		ID:    user.ID,
		Login: user.Login,
	}, nil
}
