package domain

import "errors"

var (
	ErrIncorrectUser = errors.New("incorrect user")
	ErrUserNotFound  = errors.New("user not found")
)
