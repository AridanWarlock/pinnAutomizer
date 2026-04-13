package segmentio

import (
	"github.com/segmentio/kafka-go"
)

type Headers = map[string]string

type Message struct {
	Topic     string
	Partition int
	Offset    int64
	Key       []byte
	Value     []byte

	Headers Headers
}

func NewMessage(
	topic string,
	partition int,
	offset int64,
	key []byte,
	value []byte,
	headers Headers,
) Message {
	return Message{
		Topic:     topic,
		Partition: partition,
		Offset:    offset,
		Key:       key,
		Value:     value,
		Headers:   headers,
	}
}

func NewMessageFromKafka(msg kafka.Message) Message {
	headers := make(map[string]string, len(msg.Headers))
	for _, header := range msg.Headers {
		headers[header.Key] = string(header.Value)
	}

	return NewMessage(
		msg.Topic,
		msg.Partition,
		msg.Offset,
		msg.Key,
		msg.Value,
		headers,
	)
}

func (m Message) ToKafkaMessage() kafka.Message {
	headers := make([]kafka.Header, 0, len(m.Headers))
	for key, value := range m.Headers {
		headers = append(headers, kafka.Header{
			Key:   key,
			Value: []byte(value),
		})
	}

	return kafka.Message{
		Topic:     m.Topic,
		Partition: m.Partition,
		Offset:    m.Offset,
		Key:       m.Key,
		Value:     m.Value,
		Headers:   headers,
	}
}
