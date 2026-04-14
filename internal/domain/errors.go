package domain

import "errors"

var (
	ErrTaskNotTrained = errors.New("task not trained")

	ErrInvalidJti      = errors.New("invalid jti")
	ErrInvalidAuthInfo = errors.New("invalid auth info")
)
