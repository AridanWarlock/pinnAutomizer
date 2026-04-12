package tasksAfterTrain

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/google/uuid"
	"github.com/rs/zerolog"
	"github.com/segmentio/kafka-go"
)

type Message struct {
	TaskID uuid.UUID `json:"task_id"`
}

type Consumer struct {
	usecase Usecase

	log zerolog.Logger
}

func NewConsumer(usecase Usecase, log zerolog.Logger) *Consumer {
	return &Consumer{
		usecase: usecase,
		log:     log,
	}
}

func (c *Consumer) HandleMessage(ctx context.Context, msg kafka.Message, idempotencyKey string) error {
	var message Message
	if err := json.Unmarshal(msg.Value, &message); err != nil {
		return fmt.Errorf("unmarshal message: %w", err)
	}

	input := Input{
		ID:             message.TaskID,
		IdempotencyKey: idempotencyKey,
	}

	err := c.usecase.UpdateTaskAfterTrain(ctx, input)
	if err != nil {
		return fmt.Errorf("usecase execute: %w", err)
	}
	return nil
}
