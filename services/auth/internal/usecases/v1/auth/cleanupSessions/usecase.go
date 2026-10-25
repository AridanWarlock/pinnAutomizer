package cleanupSessions

import (
	"context"
	"errors"
	"fmt"

	"github.com/AridanWarlock/pinnAutomizer/pkg/core"
	"github.com/AridanWarlock/pinnAutomizer/pkg/core/limit"
	"github.com/AridanWarlock/pinnAutomizer/pkg/errs"
)

type Postgres interface {
	GetPendingSessionRevocationJtis(
		ctx context.Context,
		after *core.Jti,
		batchSize limit.Limit,
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

func New(postgres Postgres, redis Redis) Usecase {
	return &usecase{
		postgres: postgres,
		redis:    redis,
	}
}

func (u *usecase) Cleanup(ctx context.Context, in Input) (Output, error) {
	pending, err := u.postgres.GetPendingSessionRevocationJtis(ctx, in.AfterJti, limit.MustNew(100))
	if err != nil {
		return Output{}, fmt.Errorf("list pending session revocations: %w", err)
	}
	var out Output
	var failures error
	for _, jti := range pending {
		if err := ctx.Err(); err != nil {
			return out, errors.Join(failures, err)
		}
		out.NextJti = &jti
		if err := u.revoke(ctx, jti); err != nil {
			failures = errors.Join(failures, err)
		}
	}
	return out, failures
}

func (u *usecase) revoke(ctx context.Context, jti core.Jti) error {
	err := u.redis.Delete(ctx, jti.ToRedisKey())
	if err != nil && !errors.Is(err, errs.ErrKeyNotFound) {
		return fmt.Errorf("delete pending Redis session: %w", err)
	}
	if err := u.postgres.ConfirmSessionRevocation(ctx, jti); err != nil {
		return fmt.Errorf("confirm pending session revocation: %w", err)
	}
	return nil
}
