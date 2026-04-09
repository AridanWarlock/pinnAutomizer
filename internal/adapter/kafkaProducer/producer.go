package kafkaProducer

import (
	"context"
	"time"

	"github.com/segmentio/kafka-go"
)

type Producer struct {
	writer *kafka.Writer
}

func New(cfg Config) *Producer {
	return &Producer{
		writer: &kafka.Writer{
			Addr:     kafka.TCP(cfg.Addr),
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

func (p *Producer) WriteMessages(ctx context.Context, msgs ...kafka.Message) error {
	return p.writer.WriteMessages(ctx, msgs...)
}

func (p *Producer) Close() error {
	return p.writer.Close()
}
