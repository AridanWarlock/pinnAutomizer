package segmentio

import (
	"context"
	"time"

	"github.com/segmentio/kafka-go"
)

type Writer struct {
	writer *kafka.Writer
}

func NewWriter(cfg WriterConfig) *Writer {
	return &Writer{
		writer: &kafka.Writer{
			Addr:     kafka.TCP(cfg.Broker),
			Balancer: &kafka.LeastBytes{},

			RequiredAcks: kafka.RequireAll,
			MaxAttempts:  5,

			Async:        false,
			BatchSize:    100,
			BatchTimeout: 10 * time.Millisecond,

			WriteTimeout: 10 * time.Second,
			ReadTimeout:  10 * time.Second,
		},
	}
}

func (w *Writer) WriteMessages(ctx context.Context, msgs ...Message) error {
	kafkaMsgs := make([]kafka.Message, len(msgs))
	for i, msg := range msgs {
		kafkaMsgs[i] = msg.ToKafkaMessage()
	}

	return w.writer.WriteMessages(ctx, kafkaMsgs...)
}

func (w *Writer) Close() error {
	return w.writer.Close()
}
