package domain

import (
	"fmt"
	"time"

	"github.com/google/uuid"

	"github.com/AridanWarlock/pinnAutomizer/pkg/validate"
)

type TaskStatus string

const (
	TaskStatusCreated TaskStatus = "created"
	TaskStatusInQueue TaskStatus = "in_queue"
	TaskStatusRunning TaskStatus = "running"
	TaskStatusDone    TaskStatus = "done"
	TaskStatusError   TaskStatus = "error"
)

type Task struct {
	ID          uuid.UUID `validate:"required,uuid" json:"id"`
	Name        string    `validate:"required" json:"name"`
	Description *string   `json:"description,omitempty"`

	Mode TaskMode `validate:"required,oneof=train retrain predict" json:"mode"`

	Status TaskStatus `validate:"required,oneof=created in_queue running error done" json:"status"`
	Error  *string    `json:"error,omitempty"`

	DataPath   string  `json:"data_path"`
	OutputPath string  `json:"output_path"`
	PlotPath   *string `json:"plot_path,omitempty"`

	UserID uuid.UUID `validate:"required,uuid" json:"user_id"`

	CreatedAt time.Time `validate:"required" json:"created_at"`
}

// TaskSnapshot is untrusted persistence/fixture input.
type TaskSnapshot Task

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
		PlotPath:   nil,

		UserID:    userID,
		CreatedAt: time.Now(),
	}

	if err := t.Validate(); err != nil {
		return Task{}, err
	}
	return t, nil
}

func RestoreTask(snapshot TaskSnapshot) (Task, error) {
	task := Task(snapshot)
	if err := task.Validate(); err != nil {
		return Task{}, err
	}
	return task, nil
}

func (t Task) IsStarted() bool {
	return t.Status != TaskStatusCreated
}

func (t Task) IsEnded() bool {
	switch t.Status {
	case TaskStatusError, TaskStatusDone:
		return true
	}

	return false
}

func (t Task) InQueue() bool {
	return t.Status == TaskStatusInQueue
}

func (t Task) IsRunning() bool {
	return t.Status == TaskStatusRunning
}

func (t Task) Validate() error {
	return validate.V.Struct(t)
}

// MarkRunning ignores duplicate starts and starts delivered after completion.
func (t *Task) MarkRunning() (bool, error) {
	if t.IsRunning() || t.IsEnded() {
		return false, nil
	}
	if !t.InQueue() {
		return false, ErrInvalidTaskTransition
	}
	t.Status = TaskStatusRunning
	return true, nil
}

// Complete accepts a result before the start event, but never overwrites a final result.
func (t *Task) Complete(runError *string) (bool, error) {
	if t.IsEnded() {
		return false, nil
	}
	if !t.InQueue() && !t.IsRunning() {
		return false, ErrInvalidTaskTransition
	}
	t.Error = runError
	if runError == nil {
		t.Status = TaskStatusDone
	} else {
		t.Status = TaskStatusError
	}
	return true, nil
}

func (t Task) CanDelete() bool {
	switch t.Status {
	case TaskStatusCreated, TaskStatusDone, TaskStatusError:
		return true
	default:
		return false
	}
}
