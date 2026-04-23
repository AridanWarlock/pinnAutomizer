package tasksGet

import (
	"context"

	"github.com/AridanWarlock/pinnAutomizer/pkg/core/pagination"
	"github.com/AridanWarlock/pinnAutomizer/pkg/validate"
	"github.com/AridanWarlock/pinnAutomizer/tasks/internal/domain"
)

type Input struct {
	Pagination pagination.Options
}

func (i Input) Validate() error {
	return validate.V.Struct(i)
}

type Output struct {
	TasksToEquation map[*domain.Task]domain.Equation `json:"tasks_to_equation"`
}

type Usecase interface {
	GetTasks(ctx context.Context, in Input) (Output, error)
}
