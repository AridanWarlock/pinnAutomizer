package tasksAccept

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/google/uuid"

	"github.com/AridanWarlock/pinnAutomizer/pkg/core"
	"github.com/AridanWarlock/pinnAutomizer/pkg/errs"
)

type Message struct {
	TaskID uuid.UUID `json:"task_id"`
	Mode   string    `json:"mode"`
}

type Consumer struct {
	usecase Usecase
}

func NewConsumer(usecase Usecase) *Consumer {
	return &Consumer{usecase: usecase}
}

func (c *Consumer) HandleMessage(ctx context.Context, msg core.KafkaMessage) error {
	idKey, err := msg.IdKey()
	if err != nil {
		return fmt.Errorf("%w: missing run ID", errs.ErrInvalidArgument)
	}
	runID, err := uuid.Parse(idKey.String())
	if err != nil {
		return fmt.Errorf("%w: missing or invalid run ID", errs.ErrInvalidArgument)
	}
	var message Message
	if err = json.Unmarshal(msg.Value, &message); err != nil {
		return fmt.Errorf("%w: decode solver command: %v", errs.ErrInvalidArgument, err)
	}
	in := Input{RunID: runID, TaskID: message.TaskID, Mode: message.Mode}
	if err := in.Validate(); err != nil {
		return err
	}
	_, err = c.usecase.AcceptTask(ctx, in)
	return err
}
