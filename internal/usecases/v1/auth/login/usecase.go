package authLogin

import (
	"context"
	"fmt"

	"github.com/AridanWarlock/pinnAutomizer/internal/domain"
	"github.com/AridanWarlock/pinnAutomizer/internal/errs"
	"github.com/AridanWarlock/pinnAutomizer/pkg/logger"
	"github.com/google/uuid"
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

type usecase struct {
	postgres              Postgres
	accessTokenGenerator  AccessTokenGenerator
	refreshTokenGenerator RefreshTokenGenerator
	hasher                PasswordHasher
}

func New(
	postgres Postgres,
	accessTokenGenerator AccessTokenGenerator,
	refreshTokenGenerator RefreshTokenGenerator,
	hasher PasswordHasher,
) Usecase {
	return &usecase{
		postgres:              postgres,
		accessTokenGenerator:  accessTokenGenerator,
		refreshTokenGenerator: refreshTokenGenerator,
		hasher:                hasher,
	}
}

func (u *usecase) Login(ctx context.Context, in Input) (Output, error) {
	log := logger.FromContext(ctx)
	if err := in.Validate(); err != nil {
		return Output{}, fmt.Errorf("%w: %v", errs.ErrInvalidArgument, err)
	}

	user, err := u.getValidUser(ctx, in)
	if err != nil {
		log.Info().Err(err).Msg("getting and validate user")
		return Output{}, errs.ErrInvalidCredentials
	}

	roles, err := u.postgres.GetRolesByUserID(ctx, user.ID)
	if err != nil {
		return Output{}, fmt.Errorf("getting roles by user: %w", err)
	}

	accessToken, err := u.accessTokenGenerator.Generate(user, roles)
	if err != nil {
		return Output{}, fmt.Errorf("generate access token: %w", err)
	}

	refreshToken, err := u.refreshTokenGenerator.Generate()
	if err != nil {
		return Output{}, fmt.Errorf("generate refresh token: %w", err)
	}

	session, err := domain.NewUserSession(
		user.ID,
		refreshToken.Sha256,
		refreshToken.ExpiresAt,
		in.Fingerprint,
	)
	if err != nil {
		return Output{}, fmt.Errorf("create session: %w", err)
	}

	if err = u.postgres.Login(ctx, session); err != nil {
		return Output{}, fmt.Errorf("saving session in postgres: %w", err)
	}

	refreshTokenWithSessionID := fmt.Sprintf("%s.%s", session.ID.String(), refreshToken.RandomBase64String)

	return Output{
		AccessTokenString:     string(accessToken),
		RefreshTokenString:    refreshTokenWithSessionID,
		RefreshTokenExpiresAt: refreshToken.ExpiresAt,
	}, nil
}

func (u *usecase) getValidUser(ctx context.Context, in Input) (domain.User, error) {
	user, err := u.postgres.GetUserByLogin(ctx, in.Login)
	if err != nil {
		return domain.User{}, fmt.Errorf("getting user by login from postgres: %w", err)
	}

	if err := u.hasher.CompareHashAndPassword(user.PasswordHash, in.Password); err != nil {
		return domain.User{}, fmt.Errorf("compare passwords hashes: %w", err)
	}

	return user, nil
}
