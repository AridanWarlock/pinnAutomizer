package tasksGet

import (
	"context"

	"github.com/AridanWarlock/pinnAutomizer/pinn/internal/domain"
	"github.com/AridanWarlock/pinnAutomizer/pkg/validate"

	"github.com/google/uuid"
)

type Input struct {
	IDs    []uuid.UUID `validate:"required,min=1,max=20,dive,required,uuid"`
	UserID uuid.UUID   `validate:"required,uuid"`
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
