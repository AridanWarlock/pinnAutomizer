package kafka

import (
	"context"

	"github.com/rs/zerolog"

	"github.com/AridanWarlock/pinnAutomizer/pkg/core"
	"github.com/AridanWarlock/pinnAutomizer/pkg/kafka/segmentio"
)

type Producer struct {
	producer *segmentio.Producer
	log      zerolog.Logger
}

func NewProducer(cfg ProducerConfig, log zerolog.Logger) *Producer {
	return &Producer{
		producer: segmentio.NewProducer(segmentio.ProducerConfig{
			Broker: cfg.Broker,
		}),
		log: log,
	}
}

func (w *Producer) WriteMessages(ctx context.Context, msgs ...core.KafkaMessage) error {
	messages := make([]segmentio.Message, len(msgs))
	for i, msg := range msgs {
		messages[i] = segmentio.NewProduceMessage(
			msg.Topic,
			msg.Key,
			msg.Value,
			msg.Headers,
		)
	}

	return w.producer.WriteMessages(ctx, messages...)
}

func (w *Producer) Close() error {
	return w.producer.Close()
}
