package core

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestMessageIdKeyDoesNotMutateAnotherMessage(t *testing.T) {
	firstKey, err := NewIdempotencyKey("first")
	require.NoError(t, err)
	secondKey, err := NewIdempotencyKey("second")
	require.NoError(t, err)
	first, err := NewProduceKafkaMessage("topic", nil, []byte("value"), KafkaHeaders{"trace": "trace-id"})
	require.NoError(t, err)
	first = first.WithIdKey(firstKey)
	second := first.WithIdKey(secondKey)
	key, err := first.IdKey()
	require.NoError(t, err)
	require.Equal(t, firstKey, key)
	key, err = second.IdKey()
	require.NoError(t, err)
	require.Equal(t, secondKey, key)
	require.Equal(t, "trace-id", second.Headers["trace"])
}

func TestMessageRejectsMissingIdKey(t *testing.T) {
	message, err := NewProduceKafkaMessage("topic", nil, []byte("value"), nil)
	require.NoError(t, err)
	_, err = message.IdKey()
	require.Error(t, err)
	require.NoError(t, message.ValidateOutgoing())
}

func TestWithIdKeyCreatesHeadersWithoutChangingOriginal(t *testing.T) {
	message, err := NewProduceKafkaMessage("topic", nil, []byte("value"), nil)
	require.NoError(t, err)
	key, err := NewIdempotencyKey("operation")
	require.NoError(t, err)
	result := message.WithIdKey(key)
	require.Nil(t, message.Headers)
	actual, err := result.IdKey()
	require.NoError(t, err)
	require.Equal(t, key, actual)
}

func TestMessageConstructorsRejectInvalidInputAndCopyBuffers(t *testing.T) {
	_, err := NewConsumeKafkaMessage("", 0, 0, nil, nil, nil)
	require.Error(t, err)
	_, err = NewConsumeKafkaMessage("topic", -1, 0, nil, nil, nil)
	require.Error(t, err)
	_, err = NewProduceKafkaMessage("topic", nil, nil, nil)
	require.Error(t, err)
	payload := []byte("result")
	headers := KafkaHeaders{"trace": "original"}
	message, err := NewProduceKafkaMessage("topic", nil, payload, headers)
	require.NoError(t, err)
	payload[0] = 'X'
	headers["trace"] = "changed"
	require.Equal(t, "result", string(message.Value))
	require.Equal(t, "original", message.Headers["trace"])
}
