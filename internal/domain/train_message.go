package domain

import "github.com/google/uuid"

type TrainMessage struct {
	TaskID      uuid.UUID      `json:"task_id"`
	MatFilePath string         `json:"mat_file_path"`
	Constants   map[string]any `json:"constants"`
}
