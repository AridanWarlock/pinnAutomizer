package domain

import (
	"github.com/AridanWarlock/pinnAutomizer/pkg/validate"

	"github.com/google/uuid"
)

type RunTaskMessage struct {
	TaskID uuid.UUID `validate:"required,uuid" json:"task_id"`
	Mode   TaskMode  `json:"mode"`
}

func (m RunTaskMessage) Validate() error {
	return validate.V.Struct(m)
}

func NewRunTaskMessage(task Task) (RunTaskMessage, error) {
	m := RunTaskMessage{
		TaskID: task.ID,
		Mode:   task.Mode,
	}

	if err := m.Validate(); err != nil {
		return RunTaskMessage{}, err
	}
	return m, nil
}
