package authLogin

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/AridanWarlock/pinnAutomizer/internal/domain"
	"github.com/AridanWarlock/pinnAutomizer/internal/errs"
	"github.com/AridanWarlock/pinnAutomizer/pkg/crypt"
	"github.com/google/uuid"
)

const (
	AccessTokenTtl  = 15 * time.Minute
	RefreshTokenTtl = 7 * 24 * time.Hour
)

type Postgres interface {
	GetUserByLogin(ctx context.Context, login string) (domain.User, error)
	GetRolesByUserID(ctx context.Context, userID uuid.UUID) ([]domain.Role, error)
	GetJtiByFingerprint(ctx context.Context, userID uuid.UUID, fingerprint domain.Fingerprint) (domain.Jti, error)
	Login(ctx context.Context, token domain.RefreshToken) (domain.RefreshToken, error)
}

type Redis interface {
	Set(ctx context.Context, key string, value any, ttl time.Duration) error
	Delete(ctx context.Context, key string) error
}

type TokenGenerator interface {
	GenerateAndGetClaims(userID uuid.UUID) (domain.AccessToken, domain.JwtClaims, error)
}

type PasswordHasher interface {
	CompareHashAndPassword(hash, password string) error
}

type usecase struct {
	postgres       Postgres
	redis          Redis
	tokenGenerator TokenGenerator
	hasher         PasswordHasher
}

func New(
	postgres Postgres,
	redis Redis,
	tokenGenerator TokenGenerator,
	hasher PasswordHasher,
) Usecase {
	return &usecase{
		postgres:       postgres,
		redis:          redis,
		tokenGenerator: tokenGenerator,
		hasher:         hasher,
	}
}

func (u *usecase) Login(ctx context.Context, in Input) (Output, error) {
	if err := in.Validate(); err != nil {
		return Output{}, fmt.Errorf("%w: %v", errs.ErrInvalidArgument, err)
	}

	auditInfo := domain.AuditInfoFromContext(ctx)
	fingerprint := auditInfo.Fingerprint

	userID, roles, err := u.getUserData(ctx, in)
	if err != nil {
		return Output{}, fmt.Errorf("getting user data: %w", err)
	}

	if err := u.deleteOldRedisSession(ctx, userID, fingerprint); err != nil {
		return Output{}, fmt.Errorf("delete old session: %w", err)
	}

	accessToken, claims, err := u.tokenGenerator.GenerateAndGetClaims(userID)
	if err != nil {
		return Output{}, fmt.Errorf("generate access token: %w", err)
	}
	jti := claims.Jti

	secureToken, expiresAt, err := u.generateSecureTokenAndLogin(ctx, userID, jti, auditInfo)
	if err != nil {
		return Output{}, fmt.Errorf("login with refresh token: %w", err)
	}

	err = u.createNewSessionInRedis(ctx, jti, userID, roles, fingerprint, claims.IssuedAt)
	if err != nil {
		return Output{}, fmt.Errorf("create new session: %w", err)
	}

	return Output{
		AccessToken:           accessToken,
		RefreshTokenString:    secureToken,
		RefreshTokenExpiresAt: expiresAt,
	}, nil
}

func (u *usecase) getUserData(ctx context.Context, in Input) (uuid.UUID, []domain.Role, error) {
	user, err := u.postgres.GetUserByLogin(ctx, in.Login)
	if err != nil {
		if errors.Is(err, errs.ErrNotFound) {
			return uuid.Nil, nil, errs.ErrInvalidCredentials
		}

		return uuid.Nil, nil, fmt.Errorf("get user by login from postgres: %v", err)
	}

	if err := u.hasher.CompareHashAndPassword(user.PasswordHash, in.Password); err != nil {
		return uuid.Nil, nil, errs.ErrInvalidCredentials
	}

	roles, err := u.postgres.GetRolesByUserID(ctx, user.ID)
	if err != nil {
		return uuid.Nil, nil, fmt.Errorf("get roles by user: %w", err)
	}

	return user.ID, roles, nil
}

func (u *usecase) deleteOldRedisSession(
	ctx context.Context,
	userID uuid.UUID,
	fingerprint domain.Fingerprint,
) error {
	jti, err := u.postgres.GetJtiByFingerprint(ctx, userID, fingerprint)
	if err != nil {
		if errors.Is(err, errs.ErrNotFound) {
			return nil
		}
		return fmt.Errorf("get old jti from postgres: %w", err)
	}

	if err := u.redis.Delete(ctx, jti.ToRedisKey()); err != nil {
		if errors.Is(err, errs.ErrKeyNotFound) {
			return nil
		}

		return fmt.Errorf("redis error: %w", err)
	}
	return nil
}

func (u *usecase) generateSecureTokenAndLogin(
	ctx context.Context,
	userID uuid.UUID,
	jti domain.Jti,
	auditInfo domain.AuditInfo,
) (string, time.Time, error) {
	secureToken := crypt.GenerateSecureToken()
	hash := crypt.Sha256(secureToken)

	refreshToken, err := domain.NewRefreshToken(
		hash,
		userID,
		jti,
		auditInfo.Fingerprint,
		auditInfo.Agent,
		auditInfo.IP,
		RefreshTokenTtl,
	)
	if err != nil {
		return "", time.Time{}, fmt.Errorf("create refresh token: %w", err)
	}

	refreshToken, err = u.postgres.Login(ctx, refreshToken)
	if err != nil {
		return "", time.Time{}, fmt.Errorf("saving refresh token in postgres: %w", err)
	}

	return secureToken, refreshToken.ExpiresAt, nil
}

func (u *usecase) createNewSessionInRedis(
	ctx context.Context,
	jti domain.Jti,
	userID uuid.UUID,
	roles []domain.Role,
	fingerprint domain.Fingerprint,
	issuedAt time.Time,
) error {
	redisSession, err := domain.NewRedisSession(
		userID,
		roles,
		fingerprint,
		issuedAt,
	)
	if err != nil {
		return fmt.Errorf("create redis session: %w", err)
	}

	err = u.redis.Set(ctx, jti.ToRedisKey(), redisSession, AccessTokenTtl)
	if err != nil {
		return fmt.Errorf("set session in redis: %w", err)
	}

	return nil
}
