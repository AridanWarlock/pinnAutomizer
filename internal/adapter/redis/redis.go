package redis

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/AridanWarlock/pinnAutomizer/internal/domain"

	"github.com/redis/go-redis/v9"
)

type Client struct {
	client *redis.Client
}

func New(cfg Config) (*Client, error) {
	client := redis.NewClient(&redis.Options{
		Addr:     cfg.Addr,
		Password: cfg.Password,
		DB:       cfg.DB,

		DialTimeout:  cfg.DialTimeout,
		ReadTimeout:  cfg.ReadTimeout,
		WriteTimeout: cfg.WriteTimeout,

		PoolSize:     cfg.PoolSize,
		MinIdleConns: cfg.MinIdleConnections,
		PoolTimeout:  cfg.PoolTimeout,
	})

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	_, err := client.Ping(ctx).Result()
	if err != nil {
		_ = client.Close()
		return nil, fmt.Errorf("ping redis: %w", err)
	}

	return &Client{
		client: client,
	}, nil
}

func (c *Client) Close() error {
	return c.client.Close()
}

type envelope struct {
	Status domain.IdempotencyStatus `json:"status"`
	Data   json.RawMessage          `json:"data,omitempty"`
}

func fullKey(key string) string {
	return "idemp:" + key
}

func (c *Client) Get(ctx context.Context, key string, target any) (domain.IdempotencyStatus, error) {
	val, err := c.client.Get(ctx, fullKey(key)).Bytes()
	if err == redis.Nil {
		return "", domain.ErrIdempotencyKeyNotFound
	}
	if err != nil {
		return "", fmt.Errorf("idempotency get error: %w", err)
	}

	var env envelope
	if err := json.Unmarshal(val, &env); err != nil {
		return "", fmt.Errorf("failed to unmarshal envelope: %w", err)
	}

	if len(env.Data) > 0 && target != nil {
		if err := json.Unmarshal(env.Data, &target); err != nil {
			return "", fmt.Errorf("failed to unmarshal data: %w", err)
		}
	}

	return env.Status, nil
}

func (c *Client) TryLock(
	ctx context.Context,
	key string,
	ttl time.Duration,
) (bool, error) {
	success, err := c.client.SetNX(ctx, fullKey(key), domain.IdempotencyStatusPending, ttl).Result()
	if err != nil {
		return false, fmt.Errorf("idempotency storage fail: %w", err)
	}

	return success, nil
}

func (c *Client) Set(
	ctx context.Context,
	key string,
	status domain.IdempotencyStatus,
	data any,
	ttl time.Duration,
) error {
	dataRaw, err := json.Marshal(data)
	if err != nil {
		return fmt.Errorf("failed to marshal data: %w", err)
	}
	payload, err := json.Marshal(envelope{
		Status: status,
		Data:   dataRaw,
	})
	if err != nil {
		return fmt.Errorf("failed to marshal data: %w", err)
	}

	err = c.client.Set(ctx, "idemp:"+key, payload, ttl).Err()
	if err != nil {
		return fmt.Errorf("idempotency storage fail: %w", err)
	}
	return nil
}

func (c *Client) Delete(ctx context.Context, key string) error {
	err := c.client.Del(ctx, fullKey(key)).Err()
	if err != nil {
		return fmt.Errorf("idempotency delete fail: %w", err)
	}

	return nil
}
