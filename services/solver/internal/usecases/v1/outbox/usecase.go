package outbox

import (
	"context"
	"fmt"

	"github.com/google/uuid"

	"github.com/AridanWarlock/pinnAutomizer/pkg/core"
	"github.com/AridanWarlock/pinnAutomizer/solver/internal/domain"
)

const publishBatchSize = 32

type SQLite interface {
	GetPendingEvents(ctx context.Context, limit int) ([]domain.Event, error)
	DeleteEvent(ctx context.Context, eventID uuid.UUID) error
}

// Kafka must confirm delivery synchronously. Async writers cannot drain an outbox safely.
type Kafka interface {
	WriteMessages(ctx context.Context, messages ...core.KafkaMessage) error
}

type usecase struct {
	sqlite SQLite
	kafka  Kafka
}

func New(sqlite SQLite, kafka Kafka) *usecase {
	return &usecase{sqlite: sqlite, kafka: kafka}
}

func (u *usecase) PublishPending(ctx context.Context) error {
	events, err := u.sqlite.GetPendingEvents(ctx, publishBatchSize)
	if err != nil {
		return fmt.Errorf("read solver outbox: %w", err)
	}
	for _, e := range events {
		if err = u.kafka.WriteMessages(ctx, e.Message); err != nil {
			return fmt.Errorf("publish solver event %s: %w", e.ID, err)
		}
		if err = u.sqlite.DeleteEvent(ctx, e.ID); err != nil {
			return fmt.Errorf("acknowledge solver event %s: %w", e.ID, err)
		}
	}
	return nil
}
