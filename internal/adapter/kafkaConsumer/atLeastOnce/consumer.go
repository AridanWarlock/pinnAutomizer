package kafkaAtLeastOnceConsumer

import (
	"context"
	"errors"
	"strconv"
	"time"

	"github.com/rs/zerolog"
	"github.com/segmentio/kafka-go"
	"golang.org/x/sync/errgroup"
)

var (
	ErrAtoiConvertRetryCount = errors.New("atoi convert publishInRetryTopic count")
	ErrRetryAtHeaderNotFound = errors.New("retry at header not found")
)

const (
	HeaderError          = "x-last-error"
	HeaderSource         = "x-original-topic"
	HeaderReason         = "x-dead-letter-reason"
	HeaderRetryCount     = "x-retry-count"
	HeaderRetryAt        = "x-retry-at"
	HeaderIdempotencyKey = "x-idempotency-key"
	RetryAtFormat        = time.RFC3339

	MaxRetries             = 3
	RetrySleepDurationBase = time.Second
)

type UsecaseFunc func(ctx context.Context, msg kafka.Message, idempotencyKey string) error

type Reader interface {
	FetchMessage(ctx context.Context) (kafka.Message, error)
	CommitMessages(ctx context.Context, msgs ...kafka.Message) error
}

type Writer interface {
	WriteMessages(ctx context.Context, msgs ...kafka.Message) error
}

type Consumer struct {
	writer Writer

	log zerolog.Logger
}

func New(
	writer Writer,

	log zerolog.Logger,
) *Consumer {

	return &Consumer{
		writer: writer,

		log: log.With().Str("component", "kafka_consumer").Logger(),
	}
}

func (c *Consumer) Run(ctx context.Context, topic string, handler UsecaseFunc) {
	cfg := kafka.ReaderConfig{
		Brokers:                nil,
		GroupID:                "",
		GroupTopics:            nil,
		Topic:                  topic,
		Partition:              0,
		Dialer:                 nil,
		QueueCapacity:          0,
		MinBytes:               0,
		MaxBytes:               0,
		MaxWait:                0,
		ReadBatchTimeout:       0,
		ReadLagInterval:        0,
		GroupBalancers:         nil,
		HeartbeatInterval:      0,
		CommitInterval:         0,
		PartitionWatchInterval: 0,
		WatchPartitionChanges:  false,
		SessionTimeout:         0,
		RebalanceTimeout:       0,
		JoinGroupBackoff:       0,
		RetentionTime:          0,
		StartOffset:            0,
		ReadBackoffMin:         0,
		ReadBackoffMax:         0,
		Logger:                 nil,
		ErrorLogger:            nil,
		IsolationLevel:         0,
		MaxAttempts:            0,
		OffsetOutOfRangeError:  false,
	}

	reader := kafka.NewReader(cfg)
	cfg.Topic = topic + ".retry"
	retryReader := kafka.NewReader(cfg)

	eg, ctx := errgroup.WithContext(ctx)

	eg.Go(func() error {
		return c.consumeTopic(ctx, reader, handler)
	})
	eg.Go(func() error {
		return c.consumeTopic(ctx, retryReader, func(
			ctx context.Context,
			msg kafka.Message,
			idempotencyKey string,
		) error {
			if err := waitRetryAt(ctx, msg.Headers); err != nil {
				return err
			}

			return handler(ctx, msg, idempotencyKey)
		})
	})

	go func() {
		_ = eg.Wait()
	}()
}

func (c *Consumer) consumeTopic(
	ctx context.Context,
	reader Reader,
	handler UsecaseFunc,
) error {
	for {
		err := c.fetchAndHandle(ctx, reader, handler)
		if err != nil {
			return err
		}
	}
}

func (c *Consumer) fetchAndHandle(ctx context.Context,
	reader Reader,
	handler UsecaseFunc,
) error {
	msg, err := reader.FetchMessage(ctx)
	if err != nil {
		if errors.Is(err, context.Canceled) {
			return err
		}

		c.log.Error().Err(err).Msg("kafka_consumer: reader.FetchMessage")
		return err
	}

	err = handler(ctx, msg, getIdempotencyKey(msg.Headers))
	if err != nil {
		if err = c.handleError(ctx, msg, err); err != nil {
			c.log.Error().Err(err).Msg("kafka_consumer: handleError")
			return err
		}

		return nil
	}

	err = reader.CommitMessages(ctx, msg)
	if err != nil {
		if errors.Is(err, context.Canceled) {
			return nil
		}

		c.log.Error().Err(err).Msg("kafka_consumer: reader.CommitMessages")
		return err
	}
}

func (c *Consumer) handleError(ctx context.Context, msg kafka.Message, handleErr error) error {
	var retries int
	var err error

	retriesHeader, ok := getHeader(msg.Headers, HeaderRetryCount)
	if ok {
		retries, err = strconv.Atoi(string(retriesHeader.Value))
		if err != nil {
			return ErrAtoiConvertRetryCount
		}
	}

	if retries < MaxRetries {
		return c.publishInRetryTopic(ctx, msg, handleErr, retries)
	}
	return c.publishInDlqTopic(ctx, msg, handleErr, "Max retries reached")
}

func getHeader(headers []kafka.Header, key string) (kafka.Header, bool) {
	for _, header := range headers {
		if header.Key == key {
			return header, true
		}
	}

	return kafka.Header{}, false
}

func getIdempotencyKey(headers []kafka.Header) string {
	header, _ := getHeader(headers, HeaderIdempotencyKey)
	return string(header.Value)
}

func (c *Consumer) publishInRetryTopic(
	ctx context.Context,
	msg kafka.Message,
	handleErr error,
	retries int,
) error {
	retries++
	retryAt := time.Now().Add(time.Duration(retries) * RetrySleepDurationBase)

	msg.Topic += ".retry"
	msg.Headers = updateHeader(msg.Headers, HeaderError, handleErr.Error())
	msg.Headers = updateHeader(msg.Headers, HeaderRetryCount, strconv.Itoa(retries+1))
	msg.Headers = updateHeader(msg.Headers, HeaderRetryAt, retryAt.Format(RetryAtFormat))

	return c.writer.WriteMessages(ctx, msg)
}

func updateHeader(headers []kafka.Header, key, value string) []kafka.Header {
	newHeader := kafka.Header{
		Key:   key,
		Value: []byte(value),
	}
	for i, h := range headers {
		if h.Key == key {
			headers[i] = newHeader
			return headers
		}
	}
	return append(headers, newHeader)
}

func (c *Consumer) publishInDlqTopic(
	ctx context.Context,
	msg kafka.Message,
	handleErr error,
	reason string,
) error {
	msg.Headers = updateHeader(msg.Headers, HeaderSource, msg.Topic)
	msg.Headers = updateHeader(msg.Headers, HeaderError, handleErr.Error())
	msg.Headers = updateHeader(msg.Headers, HeaderReason, reason)

	msg.Topic += ".dlq"

	return c.writer.WriteMessages(ctx, msg)
}

func waitRetryAt(ctx context.Context, headers []kafka.Header) error {
	retryAtHeader, ok := getHeader(headers, HeaderRetryAt)
	if !ok {
		return ErrRetryAtHeaderNotFound
	}

	retryAt, err := time.Parse(RetryAtFormat, string(retryAtHeader.Value))
	if err != nil {
		return err
	}

	duration := time.Until(retryAt)
	if duration <= 0 {
		return nil
	}

	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-time.After(duration):
		return nil
	}
}
