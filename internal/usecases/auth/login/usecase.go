package login

import (
	"context"
	"fmt"
	"pinnAutomizer/internal/domain"

	"github.com/google/uuid"
	"github.com/rs/zerolog"
)

type Postgres interface {
	GetUserByLogin(ctx context.Context, login string) (domain.User, error)
	GetRolesByUserID(ctx context.Context, userID uuid.UUID) ([]domain.Role, error)
	Login(ctx context.Context, session domain.UserSession) error
}

type AccessTokenGenerator interface {
	Generate(user domain.User, roles []domain.Role) (domain.AccessToken, error)
}

type RefreshTokenGenerator interface {
	Generate() (domain.RefreshToken, error)
}

type PasswordHasher interface {
	CompareHashAndPassword(hasherPassword, password string) error
}

type Usecase struct {
	postgres              Postgres
	accessTokenGenerator  AccessTokenGenerator
	refreshTokenGenerator RefreshTokenGenerator
	hasher                PasswordHasher

	log zerolog.Logger
}

var usecase *Usecase

func New(
	postgres Postgres,
	accessTokenGenerator AccessTokenGenerator,
	refreshTokenGenerator RefreshTokenGenerator,
	hasher PasswordHasher,
	log zerolog.Logger,
) *Usecase {
	uc := &Usecase{
		postgres:              postgres,
		accessTokenGenerator:  accessTokenGenerator,
		refreshTokenGenerator: refreshTokenGenerator,
		hasher:                hasher,

		log: log.With().Str("component", "usecase: auth.Login").Logger(),
	}

	usecase = uc

	return uc
}

func (u *Usecase) Login(ctx context.Context, in Input) (Output, error) {
	log := u.log.With().Ctx(ctx).Logger()

	if err := in.Validate(); err != nil {
		log.Info().Err(err).Msg("input validation error")
		return Output{}, err
	}

	user, err := u.getValidUser(ctx, in)
	if err != nil {
		log.Info().Err(err).Msg("input validation error")
		return Output{}, err
	}

	roles, err := u.postgres.GetRolesByUserID(ctx, user.ID)
	if err != nil {
		log.Info().Err(err).Msg("getting roles by user error")
		return Output{}, err
	}

	accessToken, err := u.accessTokenGenerator.Generate(user, roles)
	if err != nil {
		log.Info().Err(err).Msg("generate access token error")
		return Output{}, err
	}

	refreshToken, err := u.refreshTokenGenerator.Generate()
	if err != nil {
		log.Info().Err(err).Msg("generate refresh token error")
		return Output{}, err
	}

	session, err := domain.NewUserSession(
		user.ID,
		refreshToken.Sha256,
		refreshToken.ExpiresAt,
		in.Fingerprint,
	)

	if err != nil {
		log.Info().Err(err).Msg("create session error")
		return Output{}, err
	}

	err = u.postgres.Login(ctx, session)
	if err != nil {
		log.Error().Err(err).Msg("saving tokens in postgres error")
		return Output{}, err
	}

	refreshTokenWithSessionID := fmt.Sprintf("%s.%s", session.ID.String(), refreshToken.RandomBase64String)

	return Output{
		AccessTokenString:     string(accessToken),
		RefreshTokenString:    refreshTokenWithSessionID,
		RefreshTokenExpiresAt: refreshToken.ExpiresAt,
	}, nil
}

func (u *Usecase) getValidUser(ctx context.Context, in Input) (domain.User, error) {
	user, err := u.postgres.GetUserByLogin(ctx, in.Login)

	if err != nil {
		return domain.User{}, err
	}

	if err := u.hasher.CompareHashAndPassword(user.PasswordHash, in.Password); err != nil {
		return domain.User{}, err
	}

	return user, nil
}
