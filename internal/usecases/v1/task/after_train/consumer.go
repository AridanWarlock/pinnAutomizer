package tasks_after_train

import (
	"context"
	"encoding/json"
	"errors"

	"github.com/AridanWarlock/pinnAutomizer/internal/domain"
	"github.com/google/uuid"
	"github.com/rs/zerolog"
	"github.com/segmentio/kafka-go"
)

type Message struct {
	TaskID uuid.UUID `json:"task_id"`
}

type Service interface {
	UpdateTaskAfterTrain(ctx context.Context, in Input) error
}

type Consumer struct {
	service Service

	log zerolog.Logger
}

func NewConsumer(service Service, log zerolog.Logger) *Consumer {
	return &Consumer{
		service: service,
		log:     log,
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

	err := c.service.UpdateTaskAfterTrain(ctx, input)
	if err != nil {
		if errors.Is(err, domain.ErrAlreadyExists) {
			return nil
		}
		return err
	}
	return nil
}
