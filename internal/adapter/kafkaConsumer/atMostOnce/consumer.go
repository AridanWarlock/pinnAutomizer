package kafkaAtMostOnceConsumer

import (
	"context"
	"errors"

	"github.com/rs/zerolog"
	"github.com/segmentio/kafka-go"
	"golang.org/x/sync/errgroup"
)

type UsecaseHandler = func(ctx context.Context, data []byte)

type Reader interface {
	ReadMessage(ctx context.Context) (kafka.Message, error)
}

type Consumer struct {
	topic string

	reader Reader

	log zerolog.Logger
}

func New(topic string, reader Reader, log zerolog.Logger) *Consumer {
	return &Consumer{
		reader: reader,
		topic:  topic,
		log:    log.With().Str("component", "kafka_consumer").Logger(),
	}
}

func (c *Consumer) Run(ctx context.Context, handler UsecaseHandler) error {
	eg, ctx := errgroup.WithContext(ctx)

	eg.Go(func() error {
		return c.consume(ctx, handler)
	})

	return eg.Wait()
}

func (c *Consumer) consume(ctx context.Context, handler UsecaseHandler) error {
	for {
		msg, err := c.reader.ReadMessage(ctx)
		if err != nil {
			if errors.Is(err, context.Canceled) {
				return nil
			}

			c.log.Error().Err(err).Msg("kafka_consumer: reader.FetchMessage")
			return err
		}

		handler(ctx, msg.Value)
	}
}
