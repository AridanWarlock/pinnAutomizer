package domain

import "github.com/google/uuid"

type TrainMessage struct {
	TaskID uuid.UUID `json:"task_id"`
	Mode   TaskMode  `json:"mode"`
}
