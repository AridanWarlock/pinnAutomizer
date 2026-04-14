package consumer

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"time"

	"github.com/AridanWarlock/pinnAutomizer/pkg/adapter/kafka/segmentio"
	"github.com/rs/zerolog"
	"golang.org/x/sync/errgroup"
)

var (
	ErrHeaderNotFound     = errors.New("header not found")
	ErrMaxRetryReached    = errors.New("max retry reached")
	ErrInvalidRetryNumber = errors.New("invalid retry number")
)

const (
	HeaderLastError   = "X-Last-Error"
	HeaderSource      = "X-Original-Topic"
	HeaderReason      = "X-Dead-Letter-Reason"
	HeaderRetryNumber = "X-Retry-Number"
	HeaderRetryAt     = "X-Retry-At"
	RetryAtFormat     = time.RFC3339

	MaxRetries             = 3
	RetrySleepDurationBase = time.Second
)

type AtLeastOnceReader interface {
	FetchMessage(ctx context.Context) (segmentio.Message, error)
	CommitMessages(ctx context.Context, msgs ...segmentio.Message) error
	GetTopic() string
}

type Writer interface {
	WriteMessages(ctx context.Context, msgs ...segmentio.Message) error
}

type AtLeastOnceConsumer struct {
	dlqTopic string

	reader      AtLeastOnceReader
	retryReader AtLeastOnceReader
	writer      Writer

	log zerolog.Logger
}

func NewAtLeastOnceConsumer(
	reader AtLeastOnceReader,
	retryReader AtLeastOnceReader,
	writer Writer,

	log zerolog.Logger,
) *AtLeastOnceConsumer {
	return &AtLeastOnceConsumer{
		dlqTopic: reader.GetTopic() + ".dlq",

		reader:      reader,
		retryReader: retryReader,
		writer:      writer,

		log: log,
	}
}

func (c *AtLeastOnceConsumer) Run(ctx context.Context, handler Handler) error {
	eg, ctx := errgroup.WithContext(ctx)

	eg.Go(func() error {
		return c.consumeTopic(ctx, c.reader, handler)
	})

	eg.Go(func() error {
		handler := func(ctx context.Context, msg segmentio.Message) error {
			if err := waitRetryAt(ctx, msg.Headers); err != nil {
				return err
			}

			return handler(ctx, msg)
		}

		return c.consumeTopic(ctx, c.retryReader, handler)
	})

	return eg.Wait()
}

func (c *AtLeastOnceConsumer) consumeTopic(
	ctx context.Context,
	reader AtLeastOnceReader,
	handler Handler,
) error {
	for {
		err := c.fetchAndHandle(ctx, reader, handler)
		if err != nil {
			return err
		}
	}
}

func (c *AtLeastOnceConsumer) fetchAndHandle(ctx context.Context,
	reader AtLeastOnceReader,
	handler Handler,
) error {
	msg, err := reader.FetchMessage(ctx)
	if err != nil {
		if errors.Is(err, context.Canceled) {
			return err
		}

		c.log.Error().Err(err).Msg("kafka_consumer: reader.FetchMessage")
		return err
	}

	err = handler(ctx, msg)
	if err != nil {
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

func (c *AtLeastOnceConsumer) handleError(ctx context.Context, msg segmentio.Message, handleErr error) error {
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

func (c *AtLeastOnceConsumer) publishInRetryTopic(
	ctx context.Context,
	msg segmentio.Message,
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

	msg.Topic = c.retryReader.GetTopic()
	msg.Headers[HeaderLastError] = handleErr.Error()
	msg.Headers[HeaderRetryNumber] = strconv.Itoa(retries)
	msg.Headers[HeaderRetryAt] = retryAt.Format(RetryAtFormat)

	if err := c.writer.WriteMessages(ctx, msg); err != nil {
		return fmt.Errorf("write messages: %w", err)
	}
	return nil
}

func retriesNumFromHeaders(headers segmentio.Headers) (int, error) {
	retriesHeader, ok := headers[HeaderRetryNumber]
	if !ok {
		return 0, nil
	}

	retries, err := strconv.Atoi(retriesHeader)
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

func (c *AtLeastOnceConsumer) publishInDlqTopic(
	ctx context.Context,
	msg segmentio.Message,
	handleErr error,
	reason string,
) error {
	msg.Headers[HeaderSource] = msg.Topic
	msg.Topic = c.dlqTopic

	msg.Headers[HeaderLastError] = handleErr.Error()
	msg.Headers[HeaderReason] = reason

	if err := c.writer.WriteMessages(ctx, msg); err != nil {
		return fmt.Errorf("write messages: %w", err)
	}
	return nil
}

func waitRetryAt(ctx context.Context, headers segmentio.Headers) error {
	retryAtHeader, ok := headers[HeaderRetryAt]
	if !ok {
		return fmt.Errorf("get retry at header: %w", ErrHeaderNotFound)
	}

	retryAt, err := time.Parse(RetryAtFormat, retryAtHeader)
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
