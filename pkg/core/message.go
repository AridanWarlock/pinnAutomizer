package core

import (
	"fmt"
	"maps"
	"slices"
	"time"

	"github.com/AridanWarlock/pinnAutomizer/pkg/errs"
	"github.com/AridanWarlock/pinnAutomizer/pkg/validate"
)

const KafkaIdempotencyHeader = "X-Idempotency-Key"

type KafkaHeaders = map[string]string

type KafkaMessage struct {
	Key       []byte
	Value     []byte
	Topic     string `validate:"required"`
	Partition int    `validate:"gte=0"`
	Offset    int64  `validate:"gte=0"`
	Time      time.Time

	Headers KafkaHeaders
}

func NewConsumeKafkaMessage(
	topic string,
	partition int,
	offset int64,
	key []byte,
	value []byte,
	headers KafkaHeaders,
) (KafkaMessage, error) {
	message := KafkaMessage{
		Topic:     topic,
		Partition: partition,
		Offset:    offset,
		Key:       slices.Clone(key),
		Value:     slices.Clone(value),
		Headers:   maps.Clone(headers),
	}
	if err := message.Validate(); err != nil {
		return KafkaMessage{}, err
	}
	return message, nil
}

func NewProduceKafkaMessage(
	topic string,
	key []byte,
	value []byte,
	headers KafkaHeaders,
) (KafkaMessage, error) {
	message := KafkaMessage{
		Topic:   topic,
		Key:     slices.Clone(key),
		Value:   slices.Clone(value),
		Headers: maps.Clone(headers),
	}
	if err := message.ValidateOutgoing(); err != nil {
		return KafkaMessage{}, err
	}
	return message, nil
}

func (m KafkaMessage) Validate() error {
	if err := validate.V.Struct(m); err != nil {
		return fmt.Errorf("%w: %v", errs.ErrInvalidArgument, err)
	}
	return nil
}

func (m KafkaMessage) IdKey() (IdempotencyKey, error) {
	return NewIdempotencyKey(m.Headers[KafkaIdempotencyHeader])
}

// WithIdKey copies headers so changing one event cannot mutate another event.
func (m KafkaMessage) WithIdKey(key IdempotencyKey) (KafkaMessage, error) {
	if err := m.Validate(); err != nil {
		return KafkaMessage{}, err
	}
	if _, err := NewIdempotencyKey(key.String()); err != nil {
		return KafkaMessage{}, err
	}
	headers := make(KafkaHeaders, len(m.Headers)+1)
	for k, v := range m.Headers {
		headers[k] = v
	}
	headers[KafkaIdempotencyHeader] = key.String()
	m.Headers = headers
	return m, nil
}

func (m KafkaMessage) ValidateOutgoing() error {
	if err := m.Validate(); err != nil {
		return err
	}
	if err := validate.V.Var(m.Value, "required,min=1"); err != nil {
		return fmt.Errorf("%w: outgoing payload: %v", errs.ErrInvalidArgument, err)
	}
	if m.Partition != 0 || m.Offset != 0 || !m.Time.IsZero() {
		return fmt.Errorf("%w: outgoing message contains delivery metadata", errs.ErrInvalidArgument)
	}
	return nil
}
