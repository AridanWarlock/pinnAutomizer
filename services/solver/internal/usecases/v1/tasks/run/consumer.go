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
}

type ResponseMessage struct {
	TaskID uuid.UUID `json:"task_id"`
	Error  string    `json:"error,omitempty"`
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

	input := Input{
		TaskID: message.TaskID,
	}

	response := ResponseMessage{
		TaskID: message.TaskID,
	}

	err := c.usecase.RunTask(ctx, input)
	if err != nil {
		log.Error().Err(err).Msg("train pinn task")
		response.Error = err.Error()
	}

	responseBytes, err := json.Marshal(response)
	if err != nil {
		log.Error().Err(err).Msg("train pinn task")
		return fmt.Errorf("marshal response: %w", err)
	}

	err = c.writer.WriteMessages(ctx, core.NewKafkaMessage(
		"tasks.after.run",
		msg.Partition,
		msg.Offset,
		msg.Key,
		responseBytes,
		msg.Headers,
	))

	if err != nil {
		log.Error().Err(err).Msg("publish results of train pinn task")
		return fmt.Errorf("publish results of train: %w", err)
	}
	return nil
}
