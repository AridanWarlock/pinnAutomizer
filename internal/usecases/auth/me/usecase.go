package me

import (
	"context"
	"pinnAutomizer/internal/domain"

	"github.com/google/uuid"
	"github.com/rs/zerolog"
)

type Postgres interface {
	GetUserByID(ctx context.Context, id uuid.UUID) (domain.User, error)
}

type Usecase struct {
	postgres Postgres

	log zerolog.Logger
}

var usecase *Usecase

func New(
	postgres Postgres,
	log zerolog.Logger,
) *Usecase {
	uc := &Usecase{
		postgres: postgres,
		log:      log.With().Str("component", "usecase: auth.Me").Logger(),
	}

	usecase = uc

	return uc
}

func (u *Usecase) Me(ctx context.Context, in Input) (Output, error) {
	log := u.log.With().Ctx(ctx).Logger()

	if err := in.Validate(); err != nil {
		log.Info().Err(err).Msg("validate input error")
		return Output{}, err
	}

	user, err := u.postgres.GetUserByID(ctx, in.UserID)
	if err != nil {
		log.Error().Err(err).Msg("postgres: getting user by id error")
		return Output{}, err
	}

	return Output{
		UserID: user.ID,
		Login:  user.Login,
	}, nil
}
