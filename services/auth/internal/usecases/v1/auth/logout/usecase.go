package authLogout

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"

	"github.com/AridanWarlock/pinnAutomizer/pkg/core"
	"github.com/AridanWarlock/pinnAutomizer/pkg/errs"
	"github.com/AridanWarlock/pinnAutomizer/pkg/postgres/poolx"
)

type Postgres interface {
	GetJtiByFingerprintForUpdate(
		ctx context.Context,
		userID uuid.UUID,
		fingerprint core.Fingerprint,
	) (core.Jti, error)
	Logout(
		ctx context.Context,
		userID uuid.UUID,
		fingerprint core.Fingerprint,
	) error

	poolx.TxManager
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
	if err := u.postgres.InTransaction(ctx, u.logout); err != nil {
		return fmt.Errorf("logout transaction: %w", err)
	}
	return nil
}

func (u *usecase) logout(ctx context.Context) error {
	audit := core.MustAuditInfoFromContext(ctx)
	auth := core.MustAuthInfoFromContext(ctx)
	// Lock before Redis I/O: refresh must either finish first or lose its SQL CAS.
	currentJti, err := u.postgres.GetJtiByFingerprintForUpdate(ctx, auth.UserID, audit.Fingerprint)
	if err != nil {
		if errors.Is(err, errs.ErrNotFound) {
			return u.deleteSession(ctx, auth.Jti)
		}
		return fmt.Errorf("lock session in postgres: %w", err)
	}
	if currentJti != auth.Jti {
		if err := u.deleteSession(ctx, currentJti); err != nil {
			return err
		}
	}
	if err := u.deleteSession(ctx, auth.Jti); err != nil {
		return err
	}
	// Keep the row until Redis confirms deletion, so failure retains the current JTI.
	if err := u.postgres.Logout(ctx, auth.UserID, audit.Fingerprint); err != nil && !errors.Is(err, errs.ErrNotFound) {
		return fmt.Errorf("delete session from postgres: %w", err)
	}
	return nil
}

func (u *usecase) deleteSession(ctx context.Context, jti core.Jti) error {
	err := u.redis.Delete(ctx, jti.ToRedisKey())
	if err != nil && !errors.Is(err, errs.ErrKeyNotFound) {
		return fmt.Errorf("delete session from redis: %w", err)
	}
	return nil
}
