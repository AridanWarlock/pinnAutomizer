package cacheaside

import (
	"context"
	"errors"
	"reflect"
	"time"

	"golang.org/x/sync/singleflight"
)

var (
	ErrInvalidTarget = errors.New("target must be a non-nil pointer")
)

const DefaultL1CacheTtl = 30 * time.Second

type Cache interface {
	Get(key string) (any, bool)
	Set(key string, val any, ttl time.Duration)
	Delete(key string)
}

type Redis interface {
	Get(ctx context.Context, key string, target any) error
	Set(ctx context.Context, key string, value any, ttl time.Duration) error
	Delete(ctx context.Context, key string) error
}

type Option func(c *CacheAside)

func WithL1Ttl(ttl time.Duration) Option {
	return func(c *CacheAside) {
		c.l1Ttl = ttl
	}
}

type CacheAside struct {
	l1 Cache
	l2 Redis

	sf *singleflight.Group

	l1Ttl time.Duration
}

func NewCacheAside(l1 Cache, l2 Redis, opts ...Option) *CacheAside {
	c := &CacheAside{
		l1: l1,
		l2: l2,
		sf: &singleflight.Group{},
	}

	for _, opt := range opts {
		opt(c)
	}

	if c.l1Ttl <= 0 {
		c.l1Ttl = DefaultL1CacheTtl
	}

	return c
}

func (c *CacheAside) Get(ctx context.Context, key string, target any) error {
	if v, ok := c.l1.Get(key); ok {
		return c.decode(v, target)
	}

	val, err, _ := c.sf.Do(key, func() (any, error) {
		tmp := c.cloneType(target)

		if err := c.l2.Get(ctx, key, tmp); err != nil {
			return nil, err
		}

		result := reflect.ValueOf(tmp).Elem().Interface()
		c.l1.Set(key, result, c.l1Ttl)

		return result, nil
	})
	if err != nil {
		return err
	}

	return c.decode(val, target)
}

func (c *CacheAside) decode(src, target any) error {
	v := reflect.ValueOf(target)
	if v.Kind() != reflect.Ptr || v.IsNil() {
		return ErrInvalidTarget
	}
	v.Elem().Set(reflect.ValueOf(src))
	return nil
}

func (c *CacheAside) cloneType(target any) any {
	return reflect.New(reflect.TypeOf(target).Elem()).Interface()
}

func (c *CacheAside) Set(ctx context.Context, key string, value any, ttl time.Duration) error {
	err := c.l2.Set(ctx, key, value, ttl)
	if err != nil {
		return err
	}

	c.l1.Set(key, value, DefaultL1CacheTtl)
	return nil
}

func (c *CacheAside) Delete(ctx context.Context, key string) error {
	c.l1.Delete(key)
	return c.l2.Delete(ctx, key)
}
