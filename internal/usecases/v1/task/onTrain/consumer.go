package tasksOnTrain

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/AridanWarlock/pinnAutomizer/pkg/core"
	"github.com/AridanWarlock/pinnAutomizer/pkg/errs"
	"github.com/google/uuid"
	"github.com/rs/zerolog"
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

func (c *Consumer) HandleMessage(ctx context.Context, msg core.KafkaMessage) error {
	var message Message
	if err := json.Unmarshal(msg.Value, &message); err != nil {
		return fmt.Errorf("unmarshal message: %w", err)
	}

	input := Input{
		ID: message.TaskID,
	}

	err := c.usecase.UpdateTaskOnTrain(ctx, input)
	if err != nil {
		if errors.Is(err, errs.ErrOperationInProgress) {
			return nil
		}
		return err
	}
	return nil
}
