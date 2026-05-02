package kafka

import (
	"context"

	"github.com/AridanWarlock/pinnAutomizer/pkg/core"
	"github.com/AridanWarlock/pinnAutomizer/pkg/kafka/segmentio"
	"github.com/rs/zerolog"
)

type Writer struct {
	writer *segmentio.Writer
	log    zerolog.Logger
}

func NewWriter(cfg WriterConfig, log zerolog.Logger) *Writer {
	return &Writer{
		writer: segmentio.NewWriter(segmentio.WriterConfig{
			Broker: cfg.Broker,
		}),
		log: log,
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
