package core

import "errors"

var (
	ErrInvalidIdempotencyKey = errors.New("invalid idempotency key")
)
