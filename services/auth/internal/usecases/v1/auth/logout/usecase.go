package authLogout

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"

	"github.com/AridanWarlock/pinnAutomizer/pkg/core"
	"github.com/AridanWarlock/pinnAutomizer/pkg/errs"
)

type Postgres interface {
	RevokeSessions(
		ctx context.Context,
		userID uuid.UUID,
		fingerprint core.Fingerprint,
		requestJti core.Jti,
	) error
	GetSessionRevocationJtis(
		ctx context.Context,
		userID uuid.UUID,
		fingerprint core.Fingerprint,
	) ([]core.Jti, error)
	ConfirmSessionRevocation(ctx context.Context, jti core.Jti) error
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
	auth := core.MustAuthInfoFromContext(ctx)
	audit := core.MustAuditInfoFromContext(ctx)
	if err := u.postgres.RevokeSessions(ctx, auth.UserID, audit.Fingerprint, auth.Jti); err != nil {
		return fmt.Errorf("commit session revocation: %w", err)
	}
	return u.cleanupSessions(ctx, auth.UserID, audit.Fingerprint)
}

func (u *usecase) cleanupSessions(
	ctx context.Context,
	userID uuid.UUID,
	fingerprint core.Fingerprint,
) error {
	pending, err := u.postgres.GetSessionRevocationJtis(ctx, userID, fingerprint)
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
