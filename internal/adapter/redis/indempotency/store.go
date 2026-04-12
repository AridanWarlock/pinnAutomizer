package indempotency

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/AridanWarlock/pinnAutomizer/internal/adapter/redis"
	"github.com/AridanWarlock/pinnAutomizer/internal/domain"
)

type Store struct {
	redis   *redis.Redis
	ttl     time.Duration
	lockTtl time.Duration
}

func NewStore(client *redis.Redis, ttl time.Duration, lockTtl time.Duration) *Store {
	return &Store{
		redis:   client,
		ttl:     ttl,
		lockTtl: lockTtl,
	}
}

type envelope struct {
	Status domain.IdempotencyStatus `json:"status"`
	Data   json.RawMessage          `json:"data,omitempty"`
}

func fullKey(key string) string {
	return "idemp:" + key
}

func (s *Store) Get(ctx context.Context, key string, target any) (domain.IdempotencyStatus, error) {
	var env envelope
	err := s.redis.Get(ctx, fullKey(key), &env)

	if err != nil {
		return "", fmt.Errorf("get idempotency key: %w", err)
	}

	if target != nil {
		if err := json.Unmarshal(env.Data, &target); err != nil {
			return "", fmt.Errorf("failed to unmarshal data: %w", err)
		}
	}

	return env.Status, nil
}

func (s *Store) Set(
	ctx context.Context,
	key string,
	status domain.IdempotencyStatus,
	data any,
) error {
	env := envelope{
		Status: status,
		Data:   nil,
	}

	var err error
	if data != nil {
		env.Data, err = json.Marshal(data)
		if err != nil {
			return fmt.Errorf("failed to marshal data: %w", err)
		}
	}

	err = s.redis.Set(ctx, fullKey(key), env, s.ttl)

	if err != nil {
		return fmt.Errorf("storage idempotency: %w", err)
	}
	return nil
}

func (s *Store) TryLock(
	ctx context.Context,
	key string,
) (bool, error) {
	success, err := s.redis.TryLock(ctx, key, domain.IdempotencyStatusPending, s.lockTtl)
	if err != nil {
		return false, fmt.Errorf("try lock idempotency: %w", err)
	}

	return success, nil
}

func (s *Store) Delete(ctx context.Context, key string) error {
	err := s.redis.Delete(ctx, fullKey(key))
	if err != nil {
		return fmt.Errorf("delete idempotency: %w", err)
	}

	return nil
}
