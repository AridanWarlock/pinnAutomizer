package segmentio

import (
	"context"
	"errors"
	"fmt"
	"io"
	"time"

	kafkago "github.com/segmentio/kafka-go"

	"github.com/AridanWarlock/pinnAutomizer/pkg/errs"
)

var (
	ErrReaderClosed = errors.New("reader closed")
)

type Consumer struct {
	topic    string
	maxBytes int

	reader *kafkago.Reader
}

func NewConsumer(cfg ConsumerConfig, topic string, maxBytes int) *Consumer {
	reader := kafkago.NewReader(kafkago.ReaderConfig{
		Brokers:        []string{cfg.Broker},
		GroupID:        cfg.GroupID,
		Topic:          topic,
		StartOffset:    kafkago.FirstOffset,
		MaxWait:        10 * time.Millisecond,
		MaxBytes:       maxBytes,
		CommitInterval: 0,
	})

	r := &Consumer{
		topic:    topic,
		maxBytes: maxBytes,

		reader: reader,
	}

	return r
}

func (r *Consumer) FetchMessage(ctx context.Context) (Message, error) {
	msg, err := r.reader.FetchMessage(ctx)

	if err != nil {
		return Message{}, r.handleError(err)
	}

	return NewMessageFromKafka(msg), nil
}

func (r *Consumer) CommitMessages(ctx context.Context, msgs ...Message) error {
	kafkaMsgs := make([]kafkago.Message, len(msgs))
	for i, msg := range msgs {
		kafkaMsgs[i] = msg.ToKafkaMessage()
	}

	err := r.reader.CommitMessages(ctx, kafkaMsgs...)
	if err != nil {
		return r.handleError(err)
	}
	return nil
}

func (r *Consumer) ReadMessage(ctx context.Context) (Message, error) {
	kafkaMsg, err := r.reader.ReadMessage(ctx)

	if err != nil {
		return Message{}, r.handleError(err)
	}
	return NewMessageFromKafka(kafkaMsg), nil
}

func (r *Consumer) handleError(err error) error {
	switch {
	case errs.IsContextErr(err):
		return err

	case errors.Is(err, io.EOF):
		return ErrReaderClosed

	default:
		return fmt.Errorf("unexpected kafka error: %w", err)
	}
}

func (r *Consumer) Close() error {
	return r.reader.Close()
}

func (r *Consumer) GetTopic() string {
	return r.topic
}
