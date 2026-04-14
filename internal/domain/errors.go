package domain

import "errors"

var (
	ErrIDNotExist     = errors.New("id not exist")
	ErrTaskNotTrained = errors.New("task not trained")
	ErrAlreadyExists  = errors.New("already exists")

	ErrInvalidAccessToken = errors.New("invalid access token")
	ErrInvalidJti         = errors.New("invalid jti")
	ErrInvalidAuthInfo    = errors.New("invalid auth info")
)
