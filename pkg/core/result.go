package core

// Created distinguishes a new record from a successful replay.
type Created bool

// Changed identifies the caller that applied a durable transition.
type Changed bool

// PartialErr reports confirmed positions from the original batch.
// Implementations must return unique indices in [0, batch size).
// A violation is a programming error, not a retryable transport failure.
type PartialErr interface {
	error
	Confirmed() []int
}
