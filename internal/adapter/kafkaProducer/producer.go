package kafkaProducer

import (
	"context"

	"github.com/segmentio/kafka-go"
)

type Producer struct {
	writer *kafka.Writer
}

func New(cfg Config) *Producer {
	return &Producer{
		writer: &kafka.Writer{
			Addr:     kafka.TCP(cfg.Addr...),
			Balancer: &kafka.LeastBytes{},
		},
	}
}

func (p *Producer) WriteMessages(ctx context.Context, msgs ...kafka.Message) error {
	return p.writer.WriteMessages(ctx, msgs...)
}

func (p *Producer) Close() error {
	return p.writer.Close()
}
