package domain

import (
	"fmt"
	"time"

	"github.com/AridanWarlock/pinnAutomizer/pkg/validate"

	"github.com/google/uuid"
)

type TaskStatus string

const (
	TaskStatusCreated TaskStatus = "created"
	TaskStatusRunning TaskStatus = "running"
	TaskStatusDone    TaskStatus = "done"
	TaskStatusError   TaskStatus = "error"
)

type Task struct {
	ID          uuid.UUID `validate:"required,uuid" json:"id"`
	Name        string    `validate:"required" json:"name"`
	Description *string   `json:"description,omitempty"`

	Mode TaskMode `validate:"required,oneof=train retrain predict" json:"mode"`

	Status TaskStatus `validate:"required,oneof=created running error done" json:"status"`
	Error  *string    `json:"error,omitempty"`

	DataPath   string `json:"data_path"`
	OutputPath string `json:"output_path"`

	UserID uuid.UUID `validate:"required,uuid" json:"user_id"`

	CreatedAt time.Time `validate:"required" json:"created_at"`
}

func NewTask(
	name string,
	userID uuid.UUID,
	mode TaskMode,
	description *string,
) (Task, error) {
	id := uuid.New()

	t := Task{
		ID:          id,
		Name:        name,
		Description: description,

		Mode: mode,

		Status: TaskStatusCreated,

		DataPath:   fmt.Sprintf("/tasks_data/%s/", id.String()),
		OutputPath: fmt.Sprintf("/tasks_output/%s/", id.String()),

		UserID:    userID,
		CreatedAt: time.Now(),
	}

	if err := t.Validate(); err != nil {
		return Task{}, err
	}
	return t, nil
}

func (t Task) IsStarted() bool {
	return t.Status != TaskStatusCreated
}

func (t Task) Validate() error {
	return validate.V.Struct(t)
}
