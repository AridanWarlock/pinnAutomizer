package kafka_produce

import (
	"context"

	"github.com/rs/zerolog"
	"github.com/segmentio/kafka-go"
)

type Config struct {
	Addr []string `env:"KAFKA_WRITER_ADDR"`
}

type Producer struct {
	writer *kafka.Writer

	log zerolog.Logger
}

func New(c Config, log zerolog.Logger) *Producer {
	return &Producer{
		writer: &kafka.Writer{
			Addr:     kafka.TCP(c.Addr...),
			Balancer: &kafka.LeastBytes{},
		},
		log: log.With().Str("component", "kafka: Producer").Logger(),
	}
}

func (p *Producer) WriteMessages(ctx context.Context, msgs ...kafka.Message) error {
	return p.writer.WriteMessages(ctx, msgs...)
}

func (p *Producer) Close() {
	p.log.Info().Msg("kafka producer: closing")

	err := p.writer.Close()
	if err != nil {
		p.log.Error().Err(err).Msg("kafka producer: writer.Close")
	}

	p.log.Info().Msg("kafka producer: closed")
}
