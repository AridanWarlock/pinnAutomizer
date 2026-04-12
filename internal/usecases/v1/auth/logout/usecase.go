package authLogout

import (
	"context"
	"errors"
	"fmt"

	"github.com/AridanWarlock/pinnAutomizer/internal/domain"
	"github.com/AridanWarlock/pinnAutomizer/internal/errs"
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

	err := u.redis.Delete(ctx, auth.Jti.ToRedisKey())

	if err != nil {
		if !errors.Is(err, errs.ErrKeyNotFound) {
			return fmt.Errorf("delete session from redis: %w", err)
		}
	}

	err = u.postgres.Logout(ctx, auth.UserID, audit.Fingerprint)

	if err != nil {
		if !errors.Is(err, errs.ErrNotFound) {
			return fmt.Errorf("delete session from postgres: %w", err)
		}
	}
	return nil
}
