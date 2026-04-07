package outbox

import (
	"context"
	"fmt"
	"time"

	"github.com/AridanWarlock/pinnAutomizer/internal/domain"
	"github.com/AridanWarlock/pinnAutomizer/pkg/tx"
	"github.com/rs/zerolog"

	"github.com/google/uuid"
	"github.com/segmentio/kafka-go"
)

const (
	BatchSize = 20

	HeaderIdempotencyKey = "x-idempotency-key"
)

type Postgres interface {
	GetAvailableEvents(ctx context.Context, batchSize int) ([]domain.Event, error)
	DeleteEventsByIDs(ctx context.Context, ids []uuid.UUID) error

	tx.Wrapper
}

type Writer interface {
	WriteMessages(ctx context.Context, msgs ...kafka.Message) error
}

type Worker struct {
	postgres Postgres
	writer   Writer

	stop context.CancelFunc
	log  zerolog.Logger
}

func NewWorker(postgres Postgres, writer Writer, log zerolog.Logger) *Worker {
	ctx, stop := context.WithCancel(context.Background())

	w := &Worker{
		writer:   writer,
		postgres: postgres,
		stop:     stop,
		log:      log.With().Str("component", "outbox worker").Logger(),
	}

	go func() {
		ticker := time.NewTicker(5 * time.Second)

		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			w.ProcessEvents(ctx)
		}
	}()

	return w
}

func (w *Worker) ProcessEvents(ctx context.Context) {
	err := w.postgres.Wrap(ctx, w.getAndPublishEvents)

	if err != nil {
		w.log.Error().Err(err).Msg("process event error")
		return
	}
}

func (w *Worker) getAndPublishEvents(ctx context.Context) error {
	events, err := w.postgres.GetAvailableEvents(ctx, BatchSize)
	if err != nil {
		return fmt.Errorf("getting available events from postgres: %w", err)
	}

	msgs := make([]kafka.Message, 0, len(events))
	for _, event := range events {
		msgs = append(msgs, kafka.Message{
			Topic: event.Topic,
			Value: event.Data,
			Headers: []kafka.Header{
				{
					Key:   HeaderIdempotencyKey,
					Value: []byte(event.ID.String()),
				},
			},
		})
	}

	var deliveredEvents []uuid.UUID

	switch err := w.writer.WriteMessages(ctx, msgs...).(type) {
	case nil:
		for _, event := range events {
			deliveredEvents = append(deliveredEvents, event.ID)
		}
	case kafka.WriteErrors:
		for i, event := range events {
			if err[i] == nil {
				deliveredEvents = append(deliveredEvents, event.ID)
			}
		}
	default:
		return fmt.Errorf("write messages to kafka: %w", err)
	}

	if err = w.postgres.DeleteEventsByIDs(ctx, deliveredEvents); err != nil {
		return fmt.Errorf("deleting sent events from postgres: %w", err)
	}

	return nil
}

func (w *Worker) Close() {
	w.stop()
}
