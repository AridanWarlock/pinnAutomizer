package kafka

import (
	"context"

	"github.com/AridanWarlock/pinnAutomizer/pkg/adapter/kafka/segmentio"
	"github.com/AridanWarlock/pinnAutomizer/pkg/core"
)

type Writer struct {
	writer *segmentio.Writer
}

func NewWriter(cfg WriterConfig) *Writer {
	return &Writer{
		writer: segmentio.NewWriter(segmentio.WriterConfig{
			Broker: cfg.Broker,
		}),
	}
}

func (w *Writer) WriteMessages(ctx context.Context, msgs ...core.KafkaMessage) error {
	messages := make([]segmentio.Message, len(msgs))
	for i, msg := range msgs {
		messages[i] = segmentio.NewMessage(
			msg.Topic,
			msg.Partition,
			msg.Offset,
			msg.Key,
			msg.Value,
			msg.Headers,
		)
	}

	return w.writer.WriteMessages(ctx, messages...)
}

func (w *Writer) Close() error {
	return w.writer.Close()
}
