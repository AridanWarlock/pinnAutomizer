package authCompromised

import (
	"context"
	"errors"
	"fmt"

	"github.com/AridanWarlock/pinnAutomizer/pkg/core"
	"github.com/AridanWarlock/pinnAutomizer/pkg/errs"
	"github.com/AridanWarlock/pinnAutomizer/pkg/logger"
)

type Redis interface {
	Delete(ctx context.Context, key string) error
}

type Postgres interface {
	DeleteSessionByJti(ctx context.Context, jti core.Jti) error
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

func (u *usecase) HandleCompromisedSession(ctx context.Context, in Input) error {
	log := logger.FromContext(ctx)
	if err := u.redis.Delete(ctx, in.Jti.ToRedisKey()); err != nil {
		if !errors.Is(err, errs.ErrKeyNotFound) {
			log.Err(err).Msg("delete compromised jti from redis")
		}
	}

	err := u.postgres.DeleteSessionByJti(ctx, in.Jti)
	if err != nil {
		return fmt.Errorf("delete session from postgres: %w", err)
	}

	return nil
}
