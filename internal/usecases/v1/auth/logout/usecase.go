package authLogout

import (
	"context"
	"fmt"

	"github.com/AridanWarlock/pinnAutomizer/internal/domain"
	"github.com/google/uuid"
)

type Postgres interface {
	Logout(ctx context.Context, userID uuid.UUID, fingerprint domain.Fingerprint) error
}

type Redis interface {
	Delete(ctx context.Context, key string) error
}

type usecase struct {
	postgres Postgres
	redis    Redis
}

func New(
	postgres Postgres,
	redis Redis,
) Usecase {
	return &usecase{
		postgres: postgres,
		redis:    redis,
	}
}

func (u *usecase) Logout(ctx context.Context) error {
	audit := domain.AuditInfoFromContext(ctx)
	auth := domain.AuthInfoFromContext(ctx)

	if err := u.redis.Delete(ctx, auth.Jti.ToRedisKey()); err != nil {
		return fmt.Errorf("delete session from redis: %w", err)
	}

	if err := u.postgres.Logout(ctx, auth.UserID, audit.Fingerprint); err != nil {
		return fmt.Errorf("delete session from postgres: %w", err)
	}
	return nil
}
