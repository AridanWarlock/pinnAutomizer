package kafka_produce

import (
	"context"
	"encoding/json"
	"github.com/google/uuid"
	"github.com/segmentio/kafka-go"
	"pinnAutomizer/internal/domain"
)

type trainMessage struct {
	TaskID      uuid.UUID      `json:"task_id"`
	MatFilePath string         `json:"mat_file_path"`
	Constants   map[string]any `json:"constants"`
}

func (p *Producer) PublishTaskToTrain(ctx context.Context, task domain.Task) error {
	msg := trainMessage{
		TaskID:      task.ID,
		MatFilePath: task.TrainingDataPath,
		Constants:   task.Constants,
	}

	jsonMsg, err := json.Marshal(msg)
	if err != nil {
		return err
	}

	err = p.writer.WriteMessages(ctx, kafka.Message{
		Topic: "to-train",
		Value: jsonMsg,
	})

	return err
}
