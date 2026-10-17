package outbox

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/rs/zerolog"
	"github.com/segmentio/kafka-go"

	"github.com/AridanWarlock/pinnAutomizer/pkg/core"
	"github.com/AridanWarlock/pinnAutomizer/pkg/logger"
	"github.com/AridanWarlock/pinnAutomizer/pkg/postgres/poolx"
	"github.com/AridanWarlock/pinnAutomizer/tasks/internal/domain"
)

const (
	HeaderIdempotencyKey = "X-Idempotency-Key"
	BatchSize            = 100
)

type Postgres interface {
	GetAvailableEvents(ctx context.Context, batchSize int) ([]domain.Event, error)
	DeleteEventsByIdKeys(ctx context.Context, idKeys []core.IdempotencyKey) error
	poolx.TxManager
}

type Writer interface {
	WriteMessages(ctx context.Context, msgs ...core.KafkaMessage) error
}

type Worker struct {
	postgres Postgres
	writer   Writer

	stop context.CancelFunc
	done chan struct {
	}
	log zerolog.Logger
}

func NewWorker(postgres Postgres, writer Writer, log zerolog.Logger) *Worker {
	ctx, stop := context.WithCancel(context.Background())
	ctx = logger.WithContext(ctx, log)

	w := &Worker{
		writer:   writer,
		postgres: postgres,
		stop:     stop,
		done: make(chan struct {
		}),
		log: log.With().Str("component", "outbox worker").Logger(),
	}

	go func() {
		defer close(w.done)
		ticker := time.NewTicker(5 * time.Second)
		defer ticker.Stop()

		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				w.ProcessEvents(ctx)
			}
		}
	}()

	return w
}

func (w *Worker) ProcessEvents(ctx context.Context) {
	err := w.postgres.InTransaction(ctx, w.getAndPublishEvents)

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
	if len(events) == 0 {
		return nil
	}

	msgs := make([]core.KafkaMessage, 0, len(events))
	for _, event := range events {
		message, err := core.NewProduceKafkaMessage(event.Topic, nil, event.Data, core.KafkaHeaders{
			HeaderIdempotencyKey: event.IdKey.String(),
		})
		if err != nil {
			return err
		}
		msgs = append(msgs, message)
	}

	var deliveredEvents []core.IdempotencyKey

	publishErr := w.writer.WriteMessages(ctx, msgs...)
	var failures kafka.WriteErrors
	switch {
	case publishErr == nil:
		for _, event := range events {
			deliveredEvents = append(deliveredEvents, event.IdKey)
		}
	case errors.As(publishErr, &failures) && len(failures) == len(events):
		for i, event := range events {
			if failures[i] == nil {
				deliveredEvents = append(deliveredEvents, event.IdKey)
			}
		}
	default:
		return fmt.Errorf("write messages to kafka: %w", publishErr)
	}

	if err = w.postgres.DeleteEventsByIdKeys(ctx, deliveredEvents); err != nil {
		return fmt.Errorf("deleting sent events from postgres: %w", err)
	}

	return nil
}

func (w *Worker) Close() {
	w.stop()
	<-w.done
}
