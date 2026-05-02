package domain

import "errors"

var (
	ErrInvalidMLTask     = errors.New("invalid mltask")
	ErrInvalidMLTaskMode = errors.New("invalid mltask mode")
	ErrPinnBusy          = errors.New("pinn busy")
)
