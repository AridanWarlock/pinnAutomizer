package goRedis

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/AridanWarlock/pinnAutomizer/pkg/errs"
	"github.com/redis/go-redis/v9"
)

var (
	ErrKeyNotFound   = errors.New("key not found")
	ErrClosed        = errors.New("closed")
	ErrPoolExhausted = errors.New("pool exhausted")
)

type Client struct {
	client *redis.Client
}

func New(cfg Config) (*Client, error) {
	client := redis.NewClient(&redis.Options{
		Addr:     cfg.Addr,
		Password: cfg.Password,
		DB:       cfg.DB,

		MinRetryBackoff: 100 * time.Millisecond,
		MaxRetryBackoff: 512 * time.Millisecond,

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

func (c *Client) Get(ctx context.Context, key string, target any) error {
	val, err := c.client.Get(ctx, key).Bytes()
	if err != nil {
		return c.handleCommonErr(err)
	}

	if target != nil {
		if err := json.Unmarshal(val, &target); err != nil {
			return fmt.Errorf("failed to unmarshal: %w", err)
		}
	}

	return nil
}

func (c *Client) Set(
	ctx context.Context,
	key string,
	value any,
	ttl time.Duration,
) error {
	bytes, err := json.Marshal(value)
	if err != nil {
		return fmt.Errorf("failed to marshal: %w", err)
	}

	err = c.client.Set(ctx, key, bytes, ttl).Err()
	if err != nil {
		return c.handleCommonErr(err)
	}

	return nil
}

func (c *Client) TryLock(
	ctx context.Context,
	key string,
	value any,
	ttl time.Duration,
) (bool, error) {
	bytes, err := json.Marshal(value)
	if err != nil {
		return false, fmt.Errorf("failed to marshal: %w", err)
	}

	success, err := c.client.SetNX(ctx, key, bytes, ttl).Result()
	if err != nil {
		return false, c.handleCommonErr(err)
	}

	return success, nil
}

func (c *Client) Delete(ctx context.Context, key string) error {
	err := c.client.Del(ctx, key).Err()
	if err != nil {
		return c.handleCommonErr(err)
	}

	return nil
}

func (c *Client) handleCommonErr(err error) error {
	switch {
	case errors.Is(err, redis.Nil):
		return ErrKeyNotFound
	case errs.OneOf(err, context.DeadlineExceeded, context.Canceled):
		return err
	case errors.Is(err, redis.ErrPoolTimeout):
		return ErrPoolExhausted
	case errors.Is(err, redis.ErrClosed):
		return ErrClosed

	default:
		return fmt.Errorf("unexpected redis failure: %w", err)
	}
}

func (c *Client) Close() error {
	return c.client.Close()
}
