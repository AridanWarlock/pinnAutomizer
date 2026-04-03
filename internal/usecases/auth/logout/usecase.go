package logout

import (
	"context"

	"github.com/google/uuid"
	"github.com/rs/zerolog"
)

type Postgres interface {
	Logout(ctx context.Context, userID uuid.UUID) error
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

		log: log.With().Str("component", "usecase: auth.Logout").Logger(),
	}

	usecase = uc

	return uc
}

func (u *Usecase) Logout(ctx context.Context, in Input) error {
	log := u.log.With().Ctx(ctx).Logger()

	if err := in.Validate(); err != nil {
		log.Info().Err(err).Msg("input validation error")
		return err
	}

	if err := u.postgres.Logout(ctx, in.ID); err != nil {
		log.Error().Err(err).Msg("logout postgres error")
		return err
	}
	return nil
}
