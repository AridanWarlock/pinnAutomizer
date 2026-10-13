package domain

import "errors"

var (
	ErrTaskAlreadyStarted    = errors.New("task already started")
	ErrInvalidTaskTransition = errors.New("invalid task status transition")
	ErrInvalidTaskMode       = errors.New("invalid task mode")
)
