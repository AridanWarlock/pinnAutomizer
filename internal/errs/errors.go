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
	ErrKeyNotFound          = errors.New("key not found")
	ErrClosed               = errors.New("closed")
	ErrPoolExhausted        = errors.New("pool exhausted")
	ErrOperationInProgress  = errors.New("operation in progress")
	ErrInvalidIP            = errors.New("invalid ip")
)
