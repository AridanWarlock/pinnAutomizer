package segmentio

import (
	"context"
	"errors"
	"fmt"
	"io"

	"github.com/AridanWarlock/pinnAutomizer/pkg/errs"
	"github.com/segmentio/kafka-go"
)

var (
	ErrReaderClosed = errors.New("reader closed")
)

type Reader struct {
	topic    string
	maxBytes int

	reader *kafka.Reader
}

func NewReader(cfg ReaderConfig, topic string, maxBytes int) *Reader {
	reader := kafka.NewReader(kafka.ReaderConfig{
		Brokers:  []string{cfg.Broker},
		GroupID:  cfg.GroupID,
		Topic:    topic,
		MaxBytes: maxBytes,
	})

	r := &Reader{
		topic:    topic,
		maxBytes: maxBytes,

		reader: reader,
	}

	return r
}

func (r *Reader) FetchMessage(ctx context.Context) (Message, error) {
	msg, err := r.reader.FetchMessage(ctx)

	if err != nil {
		return Message{}, r.handleError(err)
	}

	return NewMessageFromKafka(msg), nil
}

func (r *Reader) CommitMessages(ctx context.Context, msgs ...Message) error {
	kafkaMsgs := make([]kafka.Message, len(msgs))
	for i, msg := range msgs {
		kafkaMsgs[i] = msg.ToKafkaMessage()
	}

	err := r.reader.CommitMessages(ctx, kafkaMsgs...)
	if err != nil {
		return r.handleError(err)
	}
	return nil
}

func (r *Reader) ReadMessage(ctx context.Context) (Message, error) {
	kafkaMsg, err := r.reader.ReadMessage(ctx)

	if err != nil {
		return Message{}, r.handleError(err)
	}
	return NewMessageFromKafka(kafkaMsg), nil
}

func (r *Reader) handleError(err error) error {
	switch {
	case errs.IsContextErr(err):
		return err

	case errors.Is(err, io.EOF):
		return ErrReaderClosed

	default:
		return fmt.Errorf("unexpected kafka error: %w", err)
	}
}

func (r *Reader) Close() error {
	return r.reader.Close()
}

func (r *Reader) GetTopic() string {
	return r.topic
}
