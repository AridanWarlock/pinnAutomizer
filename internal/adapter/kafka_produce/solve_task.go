package kafka_produce

import (
	"context"
	"encoding/json"
	"pinnAutomizer/internal/domain"
	"pinnAutomizer/pkg/validate"

	"github.com/google/uuid"
	"github.com/segmentio/kafka-go"
)

type solveTaskMessage struct {
	TaskID    uuid.UUID      `validate:"required,uuid" json:"task_id"`
	ModelPath string         `validate:"required" json:"model_path"`
	Constants map[string]any `json:"constants"`
}

func (d solveTaskMessage) Validate() error {
	return validate.V.Struct(d)
}

func (p *Producer) PublishTaskToSolve(ctx context.Context, task domain.Task) error {
	log := p.log.With().Ctx(ctx).Logger()

	msg := &solveTaskMessage{
		TaskID:    task.ID,
		ModelPath: task.ResultsPath,
		Constants: task.Constants,
	}
	if err := msg.Validate(); err != nil {
		log.Error().Err(err).Msg("kafka producer: msg.Validate")
		return err
	}

	jsonMsg, err := json.Marshal(msg)
	if err != nil {
		log.Error().Err(err).Msg("kafka producer: json.Marshal")
		return err
	}

	err = p.writer.WriteMessages(ctx, kafka.Message{
		Topic: "to-solve",
		Value: jsonMsg,
	})
	if err != nil {
		log.Error().Err(err).Msg("kafka producer: writer.WriteMessages")
		return err
	}

	return nil
}
