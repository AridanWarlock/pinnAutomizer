package domain

import "errors"

var (
	ErrIdempotencyKeyNotFound = errors.New("idempotency key not found")
	ErrOperationInProgress    = errors.New("operation in progress")
)

var (
	ErrIDNotExist              = errors.New("id not exist")
	ErrTaskNotTrained          = errors.New("task not trained")
	ErrUnmarshalFailed         = errors.New("unmarshal failed")
	ErrAlreadyExists           = errors.New("already exists")
	ErrParseRefreshTokenFailed = errors.New("parse refresh token failed")
	ErrRefreshTokenExpired     = errors.New("refresh token expired")
	ErrInvalidFingerprint      = errors.New("invalid fingerprint")
	ErrInvalidAccessToken      = errors.New("invalid access token")
)
