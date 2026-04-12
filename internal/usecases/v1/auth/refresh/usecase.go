package authRefresh

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/AridanWarlock/pinnAutomizer/internal/domain"
	"github.com/AridanWarlock/pinnAutomizer/internal/errs"
	authLogin "github.com/AridanWarlock/pinnAutomizer/internal/usecases/v1/auth/login"
	"github.com/AridanWarlock/pinnAutomizer/pkg/crypt"
	"github.com/google/uuid"
)

type Postgres interface {
	GetRefreshTokenByHash(ctx context.Context, hash string) (domain.RefreshToken, error)
	GetRolesByUserID(ctx context.Context, userID uuid.UUID) ([]domain.Role, error)
	RotateRefreshToken(ctx context.Context, oldHash string, newHash string, newJti domain.Jti) error
}

type Redis interface {
	Set(ctx context.Context, key string, value any, ttl time.Duration) error
	Delete(ctx context.Context, key string) error
}

type TokenGenerator interface {
	GenerateAndGetClaims(userID uuid.UUID) (domain.AccessToken, domain.JwtClaims, error)
}

type usecase struct {
	postgres       Postgres
	redis          Redis
	tokenGenerator TokenGenerator
}

func New(
	postgres Postgres,
	redis Redis,
	tokenGenerator TokenGenerator,
) Usecase {
	return &usecase{
		postgres:       postgres,
		redis:          redis,
		tokenGenerator: tokenGenerator,
	}
}

func (u *usecase) Refresh(ctx context.Context, in Input) (Output, error) {
	if err := in.Validate(); err != nil {
		return Output{}, fmt.Errorf("%w: %v", errs.ErrInvalidArgument, err)
	}

	audit := domain.AuditInfoFromContext(ctx)

	oldRefresh, err := u.getValidRefreshToken(ctx, in.RefreshTokenString, audit.Fingerprint)
	if err != nil {
		return Output{}, fmt.Errorf("get valid refresh token: %w", err)
	}
	userID := oldRefresh.UserID

	roles, err := u.postgres.GetRolesByUserID(ctx, userID)
	if err != nil {
		return Output{}, fmt.Errorf("get roles by user from postgres: %w", err)
	}

	if err := u.deleteOldSession(ctx, oldRefresh.Jti); err != nil {
		return Output{}, fmt.Errorf("delete old session: %w", err)
	}

	accessToken, claims, err := u.tokenGenerator.GenerateAndGetClaims(userID)
	if err != nil {
		return Output{}, fmt.Errorf("generate access token: %w", err)
	}
	jti := claims.Jti

	rotatedToken, err := u.rotateRefreshToken(ctx, oldRefresh.Hash, jti)
	if err != nil {
		return Output{}, fmt.Errorf("rotate token: %w", err)
	}

	err = u.setNewRedisSession(ctx, jti, userID, roles, audit.Fingerprint, claims.IssuedAt)
	if err != nil {
		return Output{}, fmt.Errorf("set session in redis: %w", err)
	}

	return Output{
		AccessToken:           accessToken,
		RefreshTokenString:    rotatedToken,
		RefreshTokenExpiresAt: oldRefresh.ExpiresAt,
	}, nil
}

func (u *usecase) getValidRefreshToken(
	ctx context.Context,
	token string,
	fingerprint domain.Fingerprint,
) (domain.RefreshToken, error) {
	hash := crypt.Sha256(token)

	refresh, err := u.postgres.GetRefreshTokenByHash(ctx, hash)
	if err != nil {
		if errors.Is(err, errs.ErrNotFound) {
			return domain.RefreshToken{}, fmt.Errorf("%w: token is expired", errs.ErrAuthorizationFailed)
		}

		return domain.RefreshToken{}, fmt.Errorf("get refresh token from postgres: %w", err)
	}

	if refresh.Fingerprint != fingerprint {
		return domain.RefreshToken{}, errs.ErrSessionIsCompromised
	}

	if refresh.ExpiresAt.Before(time.Now()) {
		return domain.RefreshToken{}, fmt.Errorf("%w: token is expired", errs.ErrAuthorizationFailed)
	}

	return refresh, nil
}

func (u *usecase) deleteOldSession(ctx context.Context, jti domain.Jti) error {
	err := u.redis.Delete(ctx, jti.ToRedisKey())

	if err != nil {
		if errors.Is(err, errs.ErrKeyNotFound) {
			return nil
		}

		return fmt.Errorf("redis delete: %w", err)
	}
	return nil
}

func (u *usecase) rotateRefreshToken(
	ctx context.Context,
	oldHash string,
	newJti domain.Jti,
) (string, error) {
	token := crypt.GenerateSecureToken()
	hash := crypt.Sha256(token)

	if err := u.postgres.RotateRefreshToken(ctx, oldHash, hash, newJti); err != nil {
		return "", fmt.Errorf("rotate in postgres: %w", err)
	}

	return token, nil
}

func (u *usecase) setNewRedisSession(
	ctx context.Context,
	jti domain.Jti,
	userID uuid.UUID,
	roles []domain.Role,
	fingerprint domain.Fingerprint,
	issuedAt time.Time,
) error {
	session, err := domain.NewRedisSession(
		userID,
		roles,
		fingerprint,
		issuedAt,
	)
	if err != nil {
		return fmt.Errorf("create redis session: %w", err)
	}

	err = u.redis.Set(ctx, jti.ToRedisKey(), session, authLogin.AccessTokenTtl)
	if err != nil {
		return fmt.Errorf("redis set: %w", err)
	}
	return nil
}
