package errs

import "errors"

var (
	ErrNotFound             = errors.New("not found")
	ErrInvalidArgument      = errors.New("invalid argument")
	ErrConflict             = errors.New("conflict")
	ErrAuthorizationFailed  = errors.New("authorization failed")
	ErrInvalidCredentials   = errors.New("invalid credentials")
	ErrEntityToLarge        = errors.New("entity to large")
	ErrSessionIsCompromised = errors.New("session is compromised")
)
