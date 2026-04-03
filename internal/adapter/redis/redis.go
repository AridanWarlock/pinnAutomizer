package redis

import (
	"context"
	"encoding/json"
	"fmt"
	"pinnAutomizer/internal/domain"
	"time"

	"github.com/redis/go-redis/v9"
)

type Config struct {
	Addr     string `env:"REDIS_ADDR" env-default:"localhost:6379"`
	Password string `env:"REDIS_PASSWORD" env-default:""`
	DB       int    `env:"REDIS_DB" env-default:"0"`

	DialTimeout  time.Duration `env:"REDIS_DIAL_TIMEOUT" env-default:"5s"`
	ReadTimeout  time.Duration `env:"REDIS_READ_TIMEOUT" env-default:"3s"`
	WriteTimeout time.Duration `env:"REDIS_WRITE_TIMEOUT" env-default:"3s"`

	PoolSize           int           `env:"REDIS_POOL_SIZE" env-default:"10"`
	MinIdleConnections int           `env:"REDIS_MIN_IDLE_CONNECTIONS" env-default:"5"`
	PoolTimeout        time.Duration `env:"REDIS_POOL_TIMEOUT" env-default:"4s"`
}

type Client struct {
	client *redis.Client
}

func New(cfg Config) *Client {
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

	return &Client{
		client: client,
	}
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
