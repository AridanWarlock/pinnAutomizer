package kafka

import (
	"errors"
	"slices"

	kafkago "github.com/segmentio/kafka-go"
)

// PartialErr reports confirmed positions from the original Kafka batch.
// Implementations must return unique indices in [0, batch size).
// A violation is a programming error, not a retryable transport failure.
type PartialErr interface {
	error
	Confirmed() []int
}

type publishError struct {
	cause     error
	confirmed []int
}

func (e *publishError) Error() string {
	return e.cause.Error()
}

func (e *publishError) Unwrap() error {
	return e.cause
}

func (e *publishError) Confirmed() []int {
	return slices.Clone(e.confirmed)
}

func deliveryError(err error, count int) error {
	var failures kafkago.WriteErrors
	if !errors.As(err, &failures) || len(failures) != count {
		return err
	}
	result := &publishError{cause: err}
	for i, failure := range failures {
		if failure == nil {
			result.confirmed = append(result.confirmed, i)
		}
	}
	return result
}
