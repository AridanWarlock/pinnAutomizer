package tasksAfterRun

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/AridanWarlock/pinnAutomizer/pkg/core"
	"github.com/google/uuid"
	"github.com/rs/zerolog"
)

type Message struct {
	TaskID uuid.UUID `json:"task_id"`
	Error  *string   `json:"error,omitempty"`
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

func (c *Consumer) HandleMessage(ctx context.Context, msg core.KafkaMessage) error {
	var message Message
	if err := json.Unmarshal(msg.Value, &message); err != nil {
		return fmt.Errorf("unmarshal message: %w", err)
	}

	input := Input{
		ID:    message.TaskID,
		Error: message.Error,
	}

	err := c.usecase.UpdateTaskAfterTrain(ctx, input)
	if err != nil {
		return fmt.Errorf("usecase execute: %w", err)
	}
	return nil
}
