package kafkaAtLeastOnceConsumer

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"time"

	"github.com/rs/zerolog"
	"github.com/segmentio/kafka-go"
	"golang.org/x/sync/errgroup"
)

var (
	ErrHeaderNotFound        = errors.New("header not found")
	ErrMaxRetryReached       = errors.New("max retry reached")
	ErrInvalidRetryNumber    = errors.New("invalid retry number")
	ErrInvalidIdempotencyKey = errors.New("invalid idempotency key")
)

const (
	HeaderLastError      = "X-Last-Error"
	HeaderSource         = "X-Original-Topic"
	HeaderReason         = "X-Dead-Letter-Reason"
	HeaderRetryNumber    = "X-Retry-Number"
	HeaderRetryAt        = "X-Retry-At"
	HeaderIdempotencyKey = "X-Idempotency-Key"
	RetryAtFormat        = time.RFC3339

	MaxRetries             = 3
	RetrySleepDurationBase = time.Second
)

type HandlerFunc func(ctx context.Context, msg kafka.Message, idempotencyKey string) error

type Reader interface {
	FetchMessage(ctx context.Context) (kafka.Message, error)
	CommitMessages(ctx context.Context, msgs ...kafka.Message) error
}

type Writer interface {
	WriteMessages(ctx context.Context, msgs ...kafka.Message) error
}

type Headers map[string]string

type Consumer struct {
	broker string

	topic      string
	retryTopic string
	dlqTopic   string

	groupID      string
	retryGroupID string

	maxBytes int

	writer Writer
	log    zerolog.Logger
}

func New(
	cfg Config,
	topic string,
	writer Writer,

	log zerolog.Logger,
) *Consumer {
	return &Consumer{
		broker: cfg.Broker,

		topic:      topic,
		retryTopic: topic + ".retry",
		dlqTopic:   topic + ".dlq",

		groupID:      cfg.GroupID,
		retryGroupID: cfg.GroupID + "-retry",

		maxBytes: 1e6, // 1Mb

		writer: writer,

		log: log,
	}
}

func (c *Consumer) Run(ctx context.Context, handler HandlerFunc) {
	reader := kafka.NewReader(kafka.ReaderConfig{
		Brokers:  []string{c.broker},
		GroupID:  c.groupID,
		Topic:    c.topic,
		MaxBytes: c.maxBytes,
	})

	retryReader := kafka.NewReader(kafka.ReaderConfig{
		Brokers:  []string{c.broker},
		GroupID:  c.retryGroupID,
		Topic:    c.retryTopic,
		MaxBytes: c.maxBytes,
	})

	eg, ctx := errgroup.WithContext(ctx)

	eg.Go(func() error {
		defer func() {
			_ = reader.Close()
		}()

		return c.consumeTopic(ctx, reader, handler)
	})

	eg.Go(func() error {
		defer func() {
			_ = retryReader.Close()
		}()

		handler := func(ctx context.Context, msg kafka.Message, idempotencyKey string) error {
			if err := waitRetryAt(ctx, msg.Headers); err != nil {
				return err
			}

			return handler(ctx, msg, idempotencyKey)
		}

		return c.consumeTopic(ctx, retryReader, handler)
	})

	if err := eg.Wait(); err != nil {
		c.log.Error().Err(err).Msg("consumers stopped with error")
	}
}

func (c *Consumer) consumeTopic(
	ctx context.Context,
	reader Reader,
	handler HandlerFunc,
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
	handler HandlerFunc,
) error {
	msg, err := reader.FetchMessage(ctx)
	if err != nil {
		if errors.Is(err, context.Canceled) {
			return err
		}

		c.log.Error().Err(err).Msg("kafka_consumer: reader.FetchMessage")
		return err
	}

	idKey, err := idempotencyKeyFromHeaders(msg.Headers)
	if err != nil {
		return fmt.Errorf("getting idempotency key from headers: %w", err)
	}

	err = handler(ctx, msg, idKey)
	if err != nil {
		c.log.Debug().Msg(fmt.Sprintf("ошибка упала в логике: %v", err))
		if err = c.handleError(ctx, msg, err); err != nil {
			c.log.Error().Err(err).Msg("kafka_consumer: handleError")
			return err
		}
	}

	err = reader.CommitMessages(ctx, msg)
	if err != nil {
		if errors.Is(err, context.Canceled) {
			return nil
		}

		c.log.Error().Err(err).Msg("kafka_consumer: reader.CommitMessages")
		return err
	}
	return nil
}

func (c *Consumer) handleError(ctx context.Context, msg kafka.Message, handleErr error) error {
	err := c.publishInRetryTopic(ctx, msg, handleErr)

	switch {
	case err == nil:
		return nil
	case errors.Is(err, ErrMaxRetryReached):
		err = c.publishInDlqTopic(ctx, msg, handleErr, "Max retries reached")
		if err != nil {
			return fmt.Errorf("unexpected publish in dlq error: %w", err)
		}

		return nil
	default:
		return fmt.Errorf("unexpected publish in retry error: %w", err)
	}
}

func getHeader(headers []kafka.Header, key string) (kafka.Header, error) {
	for _, header := range headers {
		if header.Key == key {
			return header, nil
		}
	}

	return kafka.Header{}, ErrHeaderNotFound
}

func idempotencyKeyFromHeaders(headers []kafka.Header) (string, error) {
	header, err := getHeader(headers, HeaderIdempotencyKey)
	if err != nil {
		return "", fmt.Errorf("getting header: %w", err)
	}

	idKey := string(header.Value)
	if idKey == "" {
		return "", ErrInvalidIdempotencyKey
	}

	return string(header.Value), nil
}

func (c *Consumer) publishInRetryTopic(
	ctx context.Context,
	msg kafka.Message,
	handleErr error,
) error {
	retries, err := retriesNumFromHeaders(msg.Headers)
	if err != nil {
		return fmt.Errorf("get retries number from headers: %w", err)
	}
	retries++

	if retries > MaxRetries {
		return ErrMaxRetryReached
	}

	retryAt, err := calculateRetryAt(retries)
	if err != nil {
		return fmt.Errorf("calculate retry at: %w", err)
	}

	msg = kafka.Message{
		Topic:   c.retryTopic,
		Key:     msg.Key,
		Value:   msg.Value,
		Headers: msg.Headers,
	}

	msg = updateMessageHeaders(msg, map[string]string{
		HeaderLastError:   handleErr.Error(),
		HeaderRetryNumber: strconv.Itoa(retries),
		HeaderRetryAt:     retryAt.Format(RetryAtFormat),
	})

	if err := c.writer.WriteMessages(ctx, msg); err != nil {
		return fmt.Errorf("write messages: %w", err)
	}
	return nil
}

func retriesNumFromHeaders(headers []kafka.Header) (int, error) {
	retriesHeader, err := getHeader(headers, HeaderRetryNumber)
	if err != nil {
		return 0, nil
	}

	retries, err := strconv.Atoi(string(retriesHeader.Value))
	if err != nil {
		return 0, fmt.Errorf("convert retries number from header: %w", err)
	}

	return retries, nil
}

func calculateRetryAt(retryNumber int) (time.Time, error) {
	if retryNumber < 1 || retryNumber > MaxRetries {
		return time.Time{}, fmt.Errorf(
			"%w: retry number expected from 1 to %d, actual=%d",
			ErrInvalidRetryNumber,
			MaxRetries,
			retryNumber,
		)
	}

	retrySleepDuration := RetrySleepDurationBase * time.Duration(retryNumber*retryNumber)
	return time.Now().Add(retrySleepDuration), nil
}

func updateMessageHeaders(msg kafka.Message, toUpdate Headers) kafka.Message {
	headers := make([]kafka.Header, len(msg.Headers))
	copy(headers, msg.Headers)

	updated := make(map[string]struct{}, len(toUpdate))

	for i, header := range headers {
		key := header.Key

		val, ok := toUpdate[key]
		if !ok {
			continue
		}

		headers[i].Value = []byte(val)
		updated[key] = struct{}{}
	}

	for key, val := range toUpdate {
		if _, ok := updated[key]; ok {
			continue
		}

		headers = append(headers, kafka.Header{
			Key:   key,
			Value: []byte(val),
		})
	}

	msg.Headers = headers
	return msg
}

func (c *Consumer) publishInDlqTopic(
	ctx context.Context,
	msg kafka.Message,
	handleErr error,
	reason string,
) error {
	oldTopic := msg.Topic

	msg = kafka.Message{
		Topic:   c.dlqTopic,
		Key:     msg.Key,
		Value:   msg.Value,
		Headers: msg.Headers,
	}

	msg = updateMessageHeaders(msg, map[string]string{
		HeaderSource:    oldTopic,
		HeaderLastError: handleErr.Error(),
		HeaderReason:    reason,
	})

	if err := c.writer.WriteMessages(ctx, msg); err != nil {
		return fmt.Errorf("write messages: %w", err)
	}
	return nil
}

func waitRetryAt(ctx context.Context, headers []kafka.Header) error {
	retryAtHeader, err := getHeader(headers, HeaderRetryAt)
	if err != nil {
		return fmt.Errorf("getting retry at header: %w", err)
	}

	retryAt, err := time.Parse(RetryAtFormat, string(retryAtHeader.Value))
	if err != nil {
		return fmt.Errorf("parse time from retry header: %w", err)
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
