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

func NewConsumeMessage(
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

func NewProduceMessage(
	topic string,
	key []byte,
	value []byte,
	headers Headers,
) Message {
	return Message{
		Topic:   topic,
		Key:     key,
		Value:   value,
		Headers: headers,
	}
}

func NewMessageFromKafka(msg kafka.Message) Message {
	headers := make(map[string]string, len(msg.Headers))
	for _, header := range msg.Headers {
		headers[header.Key] = string(header.Value)
	}

	return Message{
		Topic:     msg.Topic,
		Partition: msg.Partition,
		Offset:    msg.Offset,
		Key:       msg.Key,
		Value:     msg.Value,
		Headers:   headers,
	}
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
