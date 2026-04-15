package domain

import (
	"time"

	"github.com/AridanWarlock/pinnAutomizer/pkg/validate"

	"github.com/google/uuid"
)

type TaskStatus string

const (
	TaskStatusCreated  TaskStatus = "created"
	TaskStatusTraining TaskStatus = "training"
	TaskStatusDone     TaskStatus = "done"
)

type Task struct {
	ID          uuid.UUID `validate:"required,uuid" json:"id"`
	Name        string    `validate:"required" json:"name"`
	Description string

	Status    TaskStatus     `validate:"required,oneof=created training done" json:"status"`
	Constants map[string]any `validate:"required" json:"constants"`

	TrainingDataPath string `json:"training_data_path"`
	ResultsPath      string `json:"results_path"`

	UserID     uuid.UUID `validate:"required,uuid" json:"user_id"`
	EquationID uuid.UUID `validate:"required,uuid" json:"equation_id"`

	CreatedAt time.Time `validate:"required" json:"created_at"`
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
