package redis

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/AridanWarlock/pinnAutomizer/pkg/adapter/redis/goRedis"
	"github.com/AridanWarlock/pinnAutomizer/pkg/errs"
)

type Redis struct {
	client *goRedis.Client
}

func NewRedis(client *goRedis.Client) *Redis {
	return &Redis{
		client: client,
	}
}

func (r *Redis) Get(ctx context.Context, key string, target any) error {
	err := r.client.Get(ctx, key, target)

	if err != nil {
		return r.handleCommonErr(err)
	}
	return nil
}

func (r *Redis) Set(
	ctx context.Context,
	key string,
	data any,
	ttl time.Duration,
) error {
	err := r.client.Set(ctx, key, data, ttl)

	if err != nil {
		return r.handleCommonErr(err)
	}
	return nil
}

func (r *Redis) TryLock(
	ctx context.Context,
	key string,
	value any,
	ttl time.Duration,
) (bool, error) {
	success, err := r.client.TryLock(ctx, key, value, ttl)

	if err != nil {
		return false, r.handleCommonErr(err)
	}
	return success, nil
}

func (r *Redis) Delete(ctx context.Context, key string) error {
	err := r.client.Delete(ctx, key)
	if err != nil {
		return r.handleCommonErr(err)
	}

	return nil
}

func (r *Redis) handleCommonErr(err error) error {
	switch {
	case errors.Is(err, goRedis.ErrKeyNotFound):
		return errs.ErrKeyNotFound
	case errors.Is(err, goRedis.ErrClosed):
		return errs.ErrClosed
	case errors.Is(err, goRedis.ErrPoolExhausted):
		return errs.ErrPoolExhausted
	default:
		return fmt.Errorf("unexpected error: %w", err)
	}
}
