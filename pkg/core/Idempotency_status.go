package core

type IdempotencyStatus string

var (
	IdempotencyStatusPending   IdempotencyStatus = "pending"
	IdempotencyStatusCompleted IdempotencyStatus = "completed"
	IdempotencyStatusFailed    IdempotencyStatus = "failed"
)
