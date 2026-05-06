package tasksRun

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/AridanWarlock/pinnAutomizer/pkg/core"
	"github.com/AridanWarlock/pinnAutomizer/pkg/logger"
	"github.com/google/uuid"
)

type Message struct {
	TaskID uuid.UUID `json:"task_id"`
	Mode   string    `json:"mode"`
}

type OnRunMessage struct {
	TaskID uuid.UUID `json:"task_id"`
}

type ResponseMessage struct {
	TaskID uuid.UUID `json:"task_id"`
	Error  *string   `json:"error,omitempty"`
}

type Writer interface {
	WriteMessages(ctx context.Context, messages ...core.KafkaMessage) error
}

type Consumer struct {
	usecase Usecase

	writer Writer
}

func NewConsumer(usecase Usecase, writer Writer) *Consumer {
	return &Consumer{
		usecase: usecase,
		writer:  writer,
	}
}

func (c *Consumer) HandleMessage(ctx context.Context, msg core.KafkaMessage) error {
	log := logger.FromContext(ctx)

	var message Message
	if err := json.Unmarshal(msg.Value, &message); err != nil {
		return fmt.Errorf("unmarshal message: %w", err)
	}
	taskID := message.TaskID

	if err := c.writeOnRunEvent(ctx, msg, taskID); err != nil {
		log.Error().Err(err).Msg("publish event on run pinn task")
		return fmt.Errorf("publish event on run pinn task: %w", err)
	}

	input := Input{
		TaskID: taskID,
		Mode:   message.Mode,
	}

	err := c.usecase.RunTask(ctx, input)
	err = c.writeAfterRunEvent(ctx, msg, taskID, err)

	if err != nil {
		log.Error().Err(err).Msg("publish event of results pinn task")
		return fmt.Errorf("publish event of results pinn task: %w", err)
	}
	return nil
}

func (c *Consumer) writeOnRunEvent(ctx context.Context, msg core.KafkaMessage, taskID uuid.UUID) error {
	bytes, err := json.Marshal(OnRunMessage{
		TaskID: taskID,
	})
	if err != nil {
		return fmt.Errorf("marshal response: %w", err)
	}

	err = c.writer.WriteMessages(ctx, core.NewProduceKafkaMessage(
		"tasks.on.run",
		msg.Key,
		bytes,
		msg.Headers,
	))

	if err != nil {
		return fmt.Errorf("publish event: %w", err)
	}
	return nil
}

func (c *Consumer) writeAfterRunEvent(ctx context.Context, msg core.KafkaMessage, taskID uuid.UUID, usecaseErr error) error {
	event := ResponseMessage{
		TaskID: taskID,
	}
	if usecaseErr != nil {
		errText := usecaseErr.Error()
		event.Error = &errText
	}
	bytes, err := json.Marshal(event)
	if err != nil {
		return fmt.Errorf("marshal event: %w", err)
	}

	err = c.writer.WriteMessages(ctx, core.NewProduceKafkaMessage(
		"tasks.after.run",
		msg.Key,
		bytes,
		msg.Headers,
	))

	if err != nil {
		return fmt.Errorf("publish event: %w", err)
	}
	return nil
}
