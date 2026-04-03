package update_task_status_after_train

import (
	"context"
	"encoding/json"
	"errors"
	"pinnAutomizer/internal/domain"

	"github.com/google/uuid"
	"github.com/rs/zerolog"
	"github.com/segmentio/kafka-go"
)

type Message struct {
	TaskID uuid.UUID `json:"task_id"`
}

type Consumer struct {
	log zerolog.Logger
}

func NewConsumer(log zerolog.Logger) *Consumer {
	return &Consumer{
		log: log.With().Str("component", "consumer: task.UpdateTaskStatusAfterTrain").Logger(),
	}
}

func (c *Consumer) HandleMessage(ctx context.Context, msg kafka.Message, idempotencyKey string) error {
	var message Message
	if err := json.Unmarshal(msg.Value, &message); err != nil {
		return domain.ErrUnmarshalFailed
	}

	input := Input{
		ID:             message.TaskID,
		IdempotencyKey: idempotencyKey,
	}

	err := usecase.UpdateTaskStatusAfterTrain(ctx, input)
	if err != nil {
		if errors.Is(err, domain.ErrAlreadyExists) {
			return nil
		}
		return err
	}
	return nil
}
