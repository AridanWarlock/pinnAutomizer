package login

import (
	"context"
	"github.com/google/uuid"
	"github.com/rs/zerolog"
	"pinnAutomizer/internal/domain"
)

type Postgres interface {
	GetUserByLogin(ctx context.Context, login string) (*domain.User, error)
	Login(ctx context.Context, tokens domain.AuthToken) error
}

type JwtService interface {
	GenerateTokensPair(userID uuid.UUID) (domain.TokensPair, error)
}

type PasswordHasher interface {
	CompareHashAndPassword(hasherPassword, password string) error
}

type Usecase struct {
	postgres   Postgres
	jwtService JwtService
	hasher     PasswordHasher

	log zerolog.Logger
}

var usecase *Usecase

func New(
	postgres Postgres,
	jwtService JwtService,
	hasher PasswordHasher,
	log zerolog.Logger,
) *Usecase {
	uc := &Usecase{
		postgres:   postgres,
		jwtService: jwtService,
		hasher:     hasher,

		log: log.With().Str("component", "usecase: auth.Login").Logger(),
	}

	usecase = uc

	return uc
}

func (u *Usecase) Login(ctx context.Context, in Input) (Output, error) {
	log := u.log.With().Ctx(ctx).Logger()

	if err := in.Validate(); err != nil {
		log.Info().
			Err(err).
			Msg("input validation error")
		return Output{}, err
	}

	userID, err := u.getValidUser(ctx, in)
	if err != nil {
		log.Info().
			Err(err).
			Msg("input validation error")
		return Output{}, err
	}

	tokensPair, err := u.jwtService.GenerateTokensPair(userID)
	if err != nil {
		log.Error().
			Err(err).
			Msg("generate jwt tokens pair error")
		return Output{}, err
	}

	authToken, err := domain.NewAuthToken(
		userID,
		tokensPair.AccessToken.Value,
		tokensPair.RefreshToken.Value,
	)
	if err != nil {
		log.Error().
			Err(err).
			Msg("auth tokens domain model creating error")
		return Output{}, err
	}

	err = u.postgres.Login(ctx, authToken)
	if err != nil {
		log.Error().
			Err(err).
			Msg("saving tokens in postgres error")
		return Output{}, err
	}
	return Output{
		AccessToken:  tokensPair.AccessToken,
		RefreshToken: tokensPair.RefreshToken,
	}, nil
}

func (u *Usecase) getValidUser(ctx context.Context, in Input) (uuid.UUID, error) {
	user, err := u.postgres.GetUserByLogin(ctx, in.Login)

	if err != nil {
		return uuid.UUID{}, err
	}

	if err := u.hasher.CompareHashAndPassword(user.PasswordHash, in.Password); err != nil {
		return uuid.UUID{}, err
	}

	return user.ID, nil
}
