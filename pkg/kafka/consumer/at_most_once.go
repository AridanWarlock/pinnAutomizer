package consumer

import (
	"context"
	"errors"

	"github.com/AridanWarlock/pinnAutomizer/pkg/kafka/segmentio"
	"github.com/rs/zerolog"
	"golang.org/x/sync/errgroup"
)

type AtMostOnceReader interface {
	ReadMessage(ctx context.Context) (segmentio.Message, error)
}

type AtMostOnceConsumer struct {
	reader AtMostOnceReader

	log zerolog.Logger
}

func NewAtMostOnceConsumer(
	reader AtMostOnceReader,
	log zerolog.Logger,
) *AtMostOnceConsumer {
	return &AtMostOnceConsumer{
		reader: reader,

		log: log,
	}
}

func (c *AtMostOnceConsumer) Run(ctx context.Context, handler Handler) error {
	eg, ctx := errgroup.WithContext(ctx)

	eg.Go(func() error {
		return c.consume(ctx, c.reader, handler)
	})

	return eg.Wait()
}

func (c *AtMostOnceConsumer) consume(
	ctx context.Context,
	reader AtMostOnceReader,
	handler Handler,
) error {
	for {
		msg, err := reader.ReadMessage(ctx)
		if err != nil {
			if errors.Is(err, context.Canceled) {
				return nil
			}

			c.log.Error().Err(err).Msg("kafka_consumer: reader.FetchMessage")
			return err
		}

		err = handler(ctx, msg)
		if err != nil {
			c.log.Warn().Err(err).Msg("handle message error")
		}
	}
}
