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
	QueueSessionRevocation(
		ctx context.Context,
		userID uuid.UUID,
		fingerprint core.Fingerprint,
		jti core.Jti,
	) error
	GetSessionRevocations(
		ctx context.Context,
		userID uuid.UUID,
		fingerprint core.Fingerprint,
	) ([]core.Jti, error)
	ConfirmSessionRevocation(ctx context.Context, jti core.Jti) error

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
	if err := u.postgres.InTransaction(ctx, u.queueLogout); err != nil {
		return fmt.Errorf("commit session revocation: %w", err)
	}
	auth := core.MustAuthInfoFromContext(ctx)
	audit := core.MustAuditInfoFromContext(ctx)
	pending, err := u.postgres.GetSessionRevocations(ctx, auth.UserID, audit.Fingerprint)
	if err != nil {
		return fmt.Errorf("get session revocations: %w", err)
	}
	for _, jti := range pending {
		if err := u.deleteSession(ctx, jti); err != nil {
			return err
		}
	}
	return nil
}

func (u *usecase) queueLogout(ctx context.Context) error {
	audit := core.MustAuditInfoFromContext(ctx)
	auth := core.MustAuthInfoFromContext(ctx)
	currentJti, err := u.postgres.GetJtiByFingerprintForUpdate(ctx, auth.UserID, audit.Fingerprint)
	found := err == nil
	if err != nil && !errors.Is(err, errs.ErrNotFound) {
		return fmt.Errorf("lock session in postgres: %w", err)
	}
	if found && currentJti != auth.Jti {
		if err := u.postgres.QueueSessionRevocation(ctx, auth.UserID, audit.Fingerprint, currentJti); err != nil {
			return fmt.Errorf("queue current session revocation: %w", err)
		}
	}
	if err := u.postgres.QueueSessionRevocation(ctx, auth.UserID, audit.Fingerprint, auth.Jti); err != nil {
		return fmt.Errorf("queue request session revocation: %w", err)
	}
	if found {
		if err := u.postgres.Logout(ctx, auth.UserID, audit.Fingerprint); err != nil && !errors.Is(err, errs.ErrNotFound) {
			return fmt.Errorf("delete refresh from postgres: %w", err)
		}
	}
	return nil
}

func (u *usecase) deleteSession(ctx context.Context, jti core.Jti) error {
	err := u.redis.Delete(ctx, jti.ToRedisKey())
	if err != nil && !errors.Is(err, errs.ErrKeyNotFound) {
		return fmt.Errorf("delete session from redis: %w", err)
	}
	if err := u.postgres.ConfirmSessionRevocation(ctx, jti); err != nil {
		return fmt.Errorf("confirm session revocation: %w", err)
	}
	return nil
}
