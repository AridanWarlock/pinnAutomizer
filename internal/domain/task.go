package domain

import (
	"github.com/google/uuid"
	"pinnAutomizer/pkg/validate"
	"time"
)

type TaskStatus string

const (
	TaskStatusCreated  TaskStatus = "created"
	TaskStatusTraining TaskStatus = "training"
	TaskStatusDone     TaskStatus = "done"
)

type Task struct {
	ID          uuid.UUID `validate:"required, uuid"`
	Name        string    `validate:"required"`
	Description string

	Status    TaskStatus `validate:"required, oneof=created training done"`
	Constants map[string]any

	TrainingDataPath string
	ResultsPath      string

	UserID     uuid.UUID `validate:"required,uuid"`
	EquationID uuid.UUID `validate:"required,uuid"`

	CreatedAt time.Time `validate:"required"`
}

func NewTask(
	name string,
	description string,
	status TaskStatus,
	constants map[string]any,
	userID uuid.UUID,
	equationID uuid.UUID,
) (Task, error) {
	t := Task{
		ID:          uuid.New(),
		Name:        name,
		Description: description,

		Status:    status,
		Constants: constants,

		TrainingDataPath: "training/" + userID.String(),
		ResultsPath:      "",

		UserID:     userID,
		EquationID: equationID,

		CreatedAt: time.Now(),
	}

	if err := t.Validate(); err != nil {
		return Task{}, err
	}
	return t, nil
}

func (t Task) Validate() error {
	return validate.V.Struct(t)
}
