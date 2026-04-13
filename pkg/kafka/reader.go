package kafka

import (
	"context"
	"errors"
	"fmt"

	"github.com/AridanWarlock/pinnAutomizer/pkg/core"
	"github.com/AridanWarlock/pinnAutomizer/pkg/kafka/consumer"
	"github.com/AridanWarlock/pinnAutomizer/pkg/kafka/segmentio"
	"github.com/rs/zerolog"
)

var (
	ErrInvalidStrategy      = errors.New("invalid strategy")
	ErrInvalidStrategySetup = errors.New("invalid strategy setup")
	ErrHeaderNotFound       = errors.New("header not found")
)

const (
	HeaderIdempotencyKey = "X-Idempotency-Key"
)

const (
	ConsumeMessageMaxBytes = 1e6
)

type Strategy int

const (
	StrategyAtLeastOnce Strategy = iota
	StrategyAtMostOnce
)

type Handler = func(ctx context.Context, msg core.KafkaMessage) error

type Reader struct {
	strategy Strategy

	cfg   ReaderConfig
	topic string

	writer *Writer

	log zerolog.Logger
}

type Option func(consumer *Reader)

func WithWriter(writer *Writer) Option {
	return func(consumer *Reader) {
		consumer.writer = writer
	}
}

func New(
	cfg ReaderConfig,
	topic string,
	strategy Strategy,
	log zerolog.Logger,
	options ...Option,
) *Reader {
	r := &Reader{
		strategy: strategy,
		cfg:      cfg,
		topic:    topic,
		writer:   nil,
		log:      log,
	}

	for _, opt := range options {
		opt(r)
	}

	return r
}

func (r *Reader) Run(
	ctx context.Context,
	handler Handler,
) error {
	switch r.strategy {
	case StrategyAtLeastOnce:
		return r.runAtLeastOnce(ctx, handler)
	case StrategyAtMostOnce:
		return r.runAtMostOnce(ctx, handler)
	default:
		return fmt.Errorf("%w: actual=%d", ErrInvalidStrategy, r.strategy)
	}
}

func (r *Reader) runAtLeastOnce(
	ctx context.Context,
	handler Handler,
) error {
	if r.writer == nil {
		return fmt.Errorf("%w: nil writer", ErrInvalidStrategySetup)
	}

	reader := segmentio.NewReader(
		segmentio.ReaderConfig{
			Broker:  r.cfg.Broker,
			GroupID: r.cfg.GroupID,
		},
		r.topic,
		ConsumeMessageMaxBytes,
	)
	defer func() {
		if err := reader.Close(); err != nil {
			r.log.Error().Err(err).Msg("closing reader")
		}
	}()

	retryReader := segmentio.NewReader(
		segmentio.ReaderConfig{
			Broker:  r.cfg.Broker,
			GroupID: r.cfg.GroupID,
		},
		r.topic+".retry",
		ConsumeMessageMaxBytes,
	)
	defer func() {
		if err := retryReader.Close(); err != nil {
			r.log.Error().Err(err).Msg("closing retry reader")
		}
	}()

	writer := segmentio.NewWriter(segmentio.WriterConfig{
		Broker: r.cfg.Broker,
	})
	defer func() {
		if err := writer.Close(); err != nil {
			r.log.Error().Err(err).Msg("closing writer")
		}
	}()

	handlerFunc := func(ctx context.Context, msg segmentio.Message) error {
		header, ok := msg.Headers[HeaderIdempotencyKey]
		if !ok {
			return fmt.Errorf("get idempotency header: %w", ErrHeaderNotFound)
		}

		idKey, err := core.NewIdempotencyKey(header)
		if err != nil {
			return fmt.Errorf("get idempotency key from headers: %w", err)
		}

		message := core.NewKafkaMessage(
			msg.Topic,
			msg.Partition,
			msg.Offset,
			msg.Key,
			msg.Value,
			msg.Headers,
		)

		return handler(idKey.WithContext(ctx), message)
	}

	atLeastOnceConsumer := consumer.NewAtLeastOnceConsumer(
		reader,
		retryReader,
		writer,
		r.log,
	)
	return atLeastOnceConsumer.Run(ctx, handlerFunc)
}

func (r *Reader) runAtMostOnce(
	ctx context.Context,
	handler Handler,
) error {
	reader := segmentio.NewReader(
		segmentio.ReaderConfig{
			Broker:  r.cfg.Broker,
			GroupID: r.cfg.GroupID,
		},
		r.topic,
		ConsumeMessageMaxBytes,
	)
	defer func() {
		if err := reader.Close(); err != nil {
			r.log.Error().Err(err).Msg("closing reader")
		}
	}()

	handlerFunc := func(ctx context.Context, msg segmentio.Message) error {
		message := core.NewKafkaMessage(
			msg.Topic,
			msg.Partition,
			msg.Offset,
			msg.Key,
			msg.Value,
			msg.Headers,
		)

		return handler(ctx, message)
	}

	atMostOnceConsumer := consumer.NewAtMostOnceConsumer(
		reader,
		r.log,
	)

	return atMostOnceConsumer.Run(ctx, handlerFunc)
}
