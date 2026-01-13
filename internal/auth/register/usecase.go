package register

import (
	"context"
	"pinnAutomizer/internal/domain"

	"github.com/rs/zerolog"
)

type Postgres interface {
	CreateUser(ctx context.Context, user *domain.User) (*domain.User, error)
}

type PasswordHasher interface {
	HashPassword(password string) (string, error)
}

type Usecase struct {
	postgres       Postgres
	passwordHasher PasswordHasher

	log zerolog.Logger
}

var usecase *Usecase

func New(
	postgres Postgres,
	passwordHasher PasswordHasher,
	log zerolog.Logger,
) *Usecase {
	u := &Usecase{
		postgres:       postgres,
		passwordHasher: passwordHasher,

		log: log.With().Str("component", "usecase: auth.Register").Logger(),
	}

	usecase = u

	return u
}

func (u *Usecase) Register(ctx context.Context, in Input) error {
	log := u.log.With().Ctx(ctx).Logger()

	if err := in.Validate(); err != nil {
		log.Info().
			Err(err).
			Msg("input validation error")
		return err
	}

	passwordHash, err := u.passwordHasher.HashPassword(in.Password)
	if err != nil {
		log.Error().
			Err(err).
			Msg("hash password error")
		return err
	}

	user, err := domain.NewUser(in.Login, passwordHash)
	if err != nil {
		log.Error().
			Err(err).
			Msg("user domain model creating error")
		return err
	}

	user, err = u.postgres.CreateUser(ctx, user)
	if err != nil {
		log.Error().
			Err(err).
			Msg("saving user in postgres error")
		return err
	}

	return nil
}
