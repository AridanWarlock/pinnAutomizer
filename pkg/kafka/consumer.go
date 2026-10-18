package kafka

import (
	"context"
	"errors"
	"fmt"

	"github.com/rs/zerolog"

	"github.com/AridanWarlock/pinnAutomizer/pkg/core"
	"github.com/AridanWarlock/pinnAutomizer/pkg/kafka/consumer"
	"github.com/AridanWarlock/pinnAutomizer/pkg/kafka/segmentio"
	"github.com/AridanWarlock/pinnAutomizer/pkg/logger"
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

type Consumer struct {
	strategy Strategy

	cfg   ConsumerConfig
	topic string

	producer *Producer

	log zerolog.Logger
}

type Option func(consumer *Consumer)

func WithProducer(producer *Producer) Option {
	return func(consumer *Consumer) {
		consumer.producer = producer
	}
}

func NewConsumer(
	cfg ConsumerConfig,
	topic string,
	strategy Strategy,
	log zerolog.Logger,
	options ...Option,
) (*Consumer, error) {
	r := &Consumer{
		strategy: strategy,
		cfg:      cfg,
		topic:    topic,
		producer: nil,
		log:      log,
	}

	for _, opt := range options {
		opt(r)
	}

	if err := r.Validate(); err != nil {
		return nil, err
	}

	return r, nil
}

func (r *Consumer) Run(
	ctx context.Context,
	handler Handler,
) error {
	ctx = logger.WithContext(ctx, r.log)

	switch r.strategy {
	case StrategyAtLeastOnce:
		return r.runAtLeastOnce(ctx, handler)
	case StrategyAtMostOnce:
		return r.runAtMostOnce(ctx, handler)
	default:
		return fmt.Errorf("%w: actual=%d", ErrInvalidStrategy, r.strategy)
	}
}

func (r *Consumer) runAtLeastOnce(
	ctx context.Context,
	handler Handler,
) error {
	reader := segmentio.NewConsumer(
		segmentio.ConsumerConfig{
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

	retryReader := segmentio.NewConsumer(
		segmentio.ConsumerConfig{
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

	producer := segmentio.NewProducer(segmentio.ProducerConfig{
		Broker: r.cfg.Broker,
	})
	defer func() {
		if err := producer.Close(); err != nil {
			r.log.Error().Err(err).Msg("closing producer")
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

		message, err := core.NewConsumeKafkaMessage(
			msg.Topic,
			msg.Partition,
			msg.Offset,
			msg.Key,
			msg.Value,
			msg.Headers,
		)
		if err != nil {
			return err
		}

		return handler(idKey.WithContext(ctx), message)
	}

	atLeastOnceConsumer := consumer.NewAtLeastOnceConsumer(
		reader,
		retryReader,
		producer,
		r.log,
	)
	return atLeastOnceConsumer.Run(ctx, handlerFunc)
}

func (r *Consumer) runAtMostOnce(
	ctx context.Context,
	handler Handler,
) error {
	reader := segmentio.NewConsumer(
		segmentio.ConsumerConfig{
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
		message, err := core.NewConsumeKafkaMessage(
			msg.Topic,
			msg.Partition,
			msg.Offset,
			msg.Key,
			msg.Value,
			msg.Headers,
		)
		if err != nil {
			return err
		}

		return handler(ctx, message)
	}

	atMostOnceConsumer := consumer.NewAtMostOnceConsumer(
		reader,
		r.log,
	)

	return atMostOnceConsumer.Run(ctx, handlerFunc)
}

func (r *Consumer) Validate() error {
	if r.strategy == StrategyAtLeastOnce && r.producer == nil {
		return fmt.Errorf("%w: nil producer", ErrInvalidStrategySetup)
	}
	return nil
}
