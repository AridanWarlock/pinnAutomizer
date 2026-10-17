package kafka

import (
	"errors"
	"fmt"
	"testing"

	kafkago "github.com/segmentio/kafka-go"
	"github.com/stretchr/testify/require"
)

func TestDeliveryErrorExposesOnlyConfirmedPositions(t *testing.T) {
	failure := errors.New("broker rejected message")
	writerErr := kafkago.WriteErrors{nil, failure, nil}
	err := deliveryError(fmt.Errorf("wrapped: %w", writerErr), 3)
	var partial interface {
		Confirmed() []int
	}
	require.ErrorAs(t, err, &partial)
	require.Equal(t, []int{0, 2}, partial.Confirmed())
	positions := partial.Confirmed()
	positions[0] = 99
	require.Equal(t, []int{0, 2}, partial.Confirmed())
	var original kafkago.WriteErrors
	require.ErrorAs(t, err, &original)
	require.Equal(t, writerErr, original)
}

func TestDeliveryErrorDoesNotInventConfirmation(t *testing.T) {
	unknown := errors.New("connection lost")
	require.ErrorIs(t, deliveryError(unknown, 2), unknown)
	require.Nil(t, deliveryError(nil, 2))
	err := deliveryError(kafkago.WriteErrors{nil}, 2)
	var partial interface {
		Confirmed() []int
	}
	require.False(t, errors.As(err, &partial))
}
