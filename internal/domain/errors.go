package domain

import "errors"

var (
	ErrIDNotExist     = errors.New("id not exist")
	ErrTaskNotTrained = errors.New("task not trained")
	ErrAlreadyExists  = errors.New("already exists")

	ErrInvalidFingerprint = errors.New("invalid fingerprint")
	ErrInvalidAccessToken = errors.New("invalid access token")
	ErrInvalidJti         = errors.New("invalid jti")
	ErrInvalidUserAgent   = errors.New("invalid user agent")
	ErrInvalidIP          = errors.New("invalid ip")
	ErrInvalidAuthInfo    = errors.New("invalid auth info")
)
