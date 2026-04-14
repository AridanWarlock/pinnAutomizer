package core

import "errors"

var (
	ErrInvalidIdempotencyKey = errors.New("invalid idempotency key")
	ErrInvalidFingerprint    = errors.New("invalid fingerprint")
	ErrInvalidUserAgent      = errors.New("invalid user agent")
	ErrInvalidIP             = errors.New("invalid ip")
	ErrInvalidAccessToken    = errors.New("invalid access token")
)
