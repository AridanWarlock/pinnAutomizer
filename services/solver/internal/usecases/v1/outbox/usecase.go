package outbox

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"

	"github.com/AridanWarlock/pinnAutomizer/pkg/core"
	"github.com/AridanWarlock/pinnAutomizer/pkg/core/limit"
	"github.com/AridanWarlock/pinnAutomizer/solver/internal/domain"
)

const publishBatchSize = 32

var publishBatchLimit = limit.MustNew(publishBatchSize)

type SQLite interface {
	GetPendingEvents(ctx context.Context, batch limit.Limit) ([]domain.Event, error)
	DeleteEvents(ctx context.Context, eventIDs []uuid.UUID) error
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
	events, err := u.sqlite.GetPendingEvents(ctx, publishBatchLimit)
	if err != nil {
		return fmt.Errorf("read solver outbox: %w", err)
	}
	if len(events) == 0 {
		return nil
	}
	messages := make([]core.KafkaMessage, len(events))
	for i, event := range events {
		messages[i] = event.Message
	}
	publishErr := u.kafka.WriteMessages(ctx, messages...)
	ids := confirmedEvents(events, publishErr)
	if len(ids) > 0 {
		if err := u.sqlite.DeleteEvents(ctx, ids); err != nil {
			return errors.Join(publishErr, fmt.Errorf("acknowledge solver events: %w", err))
		}
	}
	return publishErr
}

func confirmedEvents(events []domain.Event, publishErr error) []uuid.UUID {
	if publishErr == nil {
		ids := make([]uuid.UUID, len(events))
		for i, event := range events {
			ids[i] = event.ID
		}
		return ids
	}
	var partial core.PartialErr
	if !errors.As(publishErr, &partial) {
		return nil
	}
	indices := partial.Confirmed()
	ids := make([]uuid.UUID, 0, len(indices))
	seen := make(map[int]bool, len(indices))
	for _, index := range indices {
		if seen[index] {
			panic("partial delivery contains a duplicate confirmation index")
		}
		seen[index] = true
		ids = append(ids, events[index].ID)
	}
	return ids
}
