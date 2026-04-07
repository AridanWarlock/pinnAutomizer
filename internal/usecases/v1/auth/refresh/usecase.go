package authRefresh

import (
	"context"
	"crypto/sha256"
	"crypto/subtle"
	"fmt"
	"strings"
	"time"

	"github.com/AridanWarlock/pinnAutomizer/internal/domain"
	"github.com/AridanWarlock/pinnAutomizer/internal/errs"
	"github.com/google/uuid"
)

type Postgres interface {
	GetUserSessionById(ctx context.Context, id uuid.UUID) (domain.UserSession, error)
	GetUserByID(ctx context.Context, id uuid.UUID) (domain.User, error)
	GetRolesByUserID(ctx context.Context, userID uuid.UUID) ([]domain.Role, error)
}

type AccessTokenGenerator interface {
	Generate(user domain.User, roles []domain.Role) (domain.AccessToken, error)
}

type usecase struct {
	postgres             Postgres
	accessTokenGenerator AccessTokenGenerator
}

func New(
	postgres Postgres,
	accessTokenGenerator AccessTokenGenerator,
) Usecase {
	return &usecase{
		postgres:             postgres,
		accessTokenGenerator: accessTokenGenerator,
	}
}

func (u *usecase) Refresh(ctx context.Context, in Input) (Output, error) {
	if err := in.Validate(); err != nil {
		return Output{}, fmt.Errorf("%w: %v", errs.ErrInvalidArgument, err)
	}

	sessionID, tokenSha256, err := parseRefreshTokenFromString(in.RefreshTokenString)
	if err != nil {
		return Output{}, fmt.Errorf("parse refresh token: %w", err)
	}

	session, err := u.postgres.GetUserSessionById(ctx, sessionID)
	if err != nil {
		return Output{}, fmt.Errorf("getting session from postgres: %w", err)
	}

	if err := validateSession(session, tokenSha256); err != nil {
		return Output{}, fmt.Errorf("validate session: %w", err)
	}

	user, err := u.postgres.GetUserByID(ctx, session.UserID)
	if err != nil {
		return Output{}, fmt.Errorf("getting user by id from postgres: %w", err)
	}

	roles, err := u.postgres.GetRolesByUserID(ctx, user.ID)
	if err != nil {
		return Output{}, fmt.Errorf("getting user roles from postgres: %w", err)
	}

	accessToken, err := u.accessTokenGenerator.Generate(user, roles)
	if err != nil {
		return Output{}, fmt.Errorf("generate access token: %w", err)
	}

	return Output{AccessToken: accessToken}, nil
}

func parseRefreshTokenFromString(token string) (uuid.UUID, []byte, error) {
	tokenSplit := strings.Split(token, ".")
	if len(tokenSplit) != 2 {
		return uuid.UUID{}, nil, domain.ErrParseRefreshTokenFailed
	}

	sessionIDString := tokenSplit[0]
	sessionID, err := uuid.Parse(sessionIDString)
	if err != nil {
		return uuid.UUID{}, nil, domain.ErrParseRefreshTokenFailed
	}

	randomBytesString := tokenSplit[1]
	sha256Bytes := sha256.Sum256([]byte(randomBytesString))

	return sessionID, sha256Bytes[:], nil
}

func validateSession(session domain.UserSession, tokenHash []byte) error {
	if subtle.ConstantTimeCompare(session.TokenSha256, tokenHash) != 1 {
		return fmt.Errorf("compare refresh token hashes: %w", domain.ErrSessionCompromised)
	}

	if session.ExpiresAt.Before(time.Now()) {
		return domain.ErrRefreshTokenExpired
	}

	return nil
}
