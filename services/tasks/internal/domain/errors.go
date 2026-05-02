package domain

import "errors"

var (
	ErrTaskAlreadyStarted = errors.New("task already started")
	ErrInvalidTaskMode    = errors.New("invalid task mode")
)
