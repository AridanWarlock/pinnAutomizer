package refresh

import (
	"context"
	"pinnAutomizer/internal/domain"

	"github.com/go-playground/validator/v10"
	"github.com/google/uuid"
	"github.com/rs/zerolog"
)

type Postgres interface {
	Refresh(ctx context.Context, userID uuid.UUID, newAccessToken string) error
}

type JwtService interface {
	ValidateRefreshToken(ctx context.Context, refreshToken string) (uuid.UUID, error)
	GenerateAccessToken(userID uuid.UUID) (domain.Token, error)
}

type Usecase struct {
	postgres   Postgres
	jwtService JwtService

	log      zerolog.Logger
	validate *validator.Validate
}

var usecase *Usecase

func New(
	postgres Postgres,
	jwtService JwtService,
	log zerolog.Logger,
	validate *validator.Validate,
) *Usecase {
	uc := &Usecase{
		postgres:   postgres,
		jwtService: jwtService,

		log:      log.With().Str("component", "usecase: auth.Refresh").Logger(),
		validate: validate,
	}

	usecase = uc

	return uc
}

func (u *Usecase) Refresh(ctx context.Context, in Input) (Output, error) {
	log := u.log.With().Ctx(ctx).Logger()

	if err := in.Validate(u.validate); err != nil {
		log.Info().
			Err(err).
			Msg("input validation error")
		return Output{}, err
	}

	userID, err := u.jwtService.ValidateRefreshToken(ctx, in.RefreshToken)
	if err != nil {
		log.Info().
			Err(err).
			Msg("jwt refresh token validating error")
		return Output{}, err
	}

	accessToken, err := u.jwtService.GenerateAccessToken(userID)
	if err != nil {
		log.Error().
			Err(err).
			Msg("jwt access token generating error")
		return Output{}, err
	}

	err = u.postgres.Refresh(ctx, userID, accessToken.Value)
	if err != nil {
		log.Error().
			Err(err).
			Msg("refresh token postgres error")
		return Output{}, err
	}

	return Output{AccessToken: accessToken}, nil
}
