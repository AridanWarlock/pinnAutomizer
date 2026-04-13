package core

import (
	"time"
)

type KafkaHeaders = map[string]string

type KafkaMessage struct {
	Key       []byte
	Value     []byte
	Topic     string
	Partition int
	Offset    int64
	Time      time.Time

	Headers KafkaHeaders
}

func NewKafkaMessage(
	topic string,
	partition int,
	offset int64,
	key []byte,
	value []byte,
	headers KafkaHeaders,
) KafkaMessage {
	return KafkaMessage{
		Topic:     topic,
		Partition: partition,
		Offset:    offset,
		Key:       key,
		Value:     value,
		Headers:   headers,
	}
}
