package authCompromised

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/AridanWarlock/pinnAutomizer/pkg/core"
	"github.com/google/uuid"
)

type Message struct {
	Jti uuid.UUID `json:"jti"`
}

type Consumer struct {
	usecase Usecase
}

func NewConsumer(usecase Usecase) *Consumer {
	return &Consumer{
		usecase: usecase,
	}
}

func (c *Consumer) HandleMessage(ctx context.Context, msg core.KafkaMessage) error {
	var message Message
	if err := json.Unmarshal(msg.Value, &message); err != nil {
		return fmt.Errorf("unmarshal message: %w", err)
	}

	jti, err := core.NewJti(message.Jti)
	if err != nil {
		return fmt.Errorf("invalid jti in message: %w", err)
	}

	err = c.usecase.HandleCompromisedSession(ctx, Input{
		Jti: jti,
	})

	if err != nil {
		return fmt.Errorf("handle compromised session: %w", err)
	}
	return nil
}
