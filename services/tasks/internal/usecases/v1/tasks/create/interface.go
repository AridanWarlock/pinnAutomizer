package tasksCreate

import (
	"context"

	"github.com/AridanWarlock/pinnAutomizer/pkg/validate"
	"github.com/AridanWarlock/pinnAutomizer/tasks/internal/domain"
)

type Input struct {
	Name        string `validate:"required"`
	Description *string

	Mode domain.TaskMode

	Files []domain.TaskFile
}

func (i Input) Validate() error {
	return validate.V.Struct(i)
}

type Output struct {
	Task domain.Task `json:"task"`
}

type Usecase interface {
	CreateTask(ctx context.Context, in Input) (Output, error)
}
