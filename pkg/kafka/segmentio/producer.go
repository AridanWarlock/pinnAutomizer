package segmentio

import (
	"context"
	"time"

	kafkago "github.com/segmentio/kafka-go"
)

type Producer struct {
	writer *kafkago.Writer
}

func NewProducer(cfg ProducerConfig) *Producer {
	return &Producer{
		writer: &kafkago.Writer{
			Addr: kafkago.TCP(cfg.Broker),

			RequiredAcks: kafkago.RequireAll,
			MaxAttempts:  5,

			Async:        false,
			BatchSize:    100,
			BatchTimeout: 10 * time.Millisecond,
		},
	}
}

func (w *Producer) WriteMessages(ctx context.Context, msgs ...Message) error {
	kafkaMsgs := make([]kafkago.Message, len(msgs))
	for i, msg := range msgs {
		kafkaMsgs[i] = msg.ToKafkaMessage()
	}

	return w.writer.WriteMessages(ctx, kafkaMsgs...)
}

func (w *Producer) Close() error {
	return w.writer.Close()
}
