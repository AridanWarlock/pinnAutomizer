package domain

import (
	"github.com/AridanWarlock/pinnAutomizer/pkg/validate"

	"github.com/google/uuid"
)

type SolveTaskMessage struct {
	TaskID    uuid.UUID      `validate:"required,uuid" json:"task_id"`
	ModelPath string         `validate:"required" json:"model_path"`
	Constants map[string]any `json:"constants"`
}

func (m SolveTaskMessage) Validate() error {
	return validate.V.Struct(m)
}

func NewSolveTaskMessage(task Task) (SolveTaskMessage, error) {
	m := SolveTaskMessage{
		TaskID:    task.ID,
		ModelPath: task.ResultsPath,
		Constants: task.Constants,
	}

	if err := m.Validate(); err != nil {
		return SolveTaskMessage{}, err
	}
	return m, nil
}
