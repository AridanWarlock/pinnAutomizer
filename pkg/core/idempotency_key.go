package core

import "context"

type idempotencyKeyKey struct{}

type IdempotencyKey string

func NewIdempotencyKey(key string) (IdempotencyKey, error) {
	k := IdempotencyKey(key)

	if err := k.Validate(); err != nil {
		return "", err
	}
	return k, nil
}

func (k IdempotencyKey) Validate() error {
	if k == "" {
		return ErrInvalidIdempotencyKey
	}
	return nil
}

func (k IdempotencyKey) ToRedisKey() string {
	return string("idemp:" + k)
}

func (k IdempotencyKey) WithContext(ctx context.Context) context.Context {
	return context.WithValue(ctx, idempotencyKeyKey{}, k)
}

func IdempotencyKeyFromContext(ctx context.Context) (IdempotencyKey, bool) {
	v, ok := ctx.Value(idempotencyKeyKey{}).(IdempotencyKey)
	return v, ok
}

func MustIdempotencyKeyFromContext(ctx context.Context) IdempotencyKey {
	k, ok := IdempotencyKeyFromContext(ctx)
	if !ok {
		panic("no idempotency key in context")
	}
	return k
}
