package tasksCreate

import (
	"context"

	"github.com/AridanWarlock/pinnAutomizer/pkg/validate"
	"github.com/AridanWarlock/pinnAutomizer/tasks/internal/domain"

	"github.com/google/uuid"
)

type Input struct {
	Name         string `validate:"required"`
	Description  string
	Constants    map[string]any
	UserID       uuid.UUID `validate:"required,uuid"`
	EquationType string    `validate:"required,oneof=heat wave"`
}

func (i Input) Validate() error {
	return validate.V.Struct(i)
}

type Output struct {
	Task     domain.Task     `json:"task"`
	Equation domain.Equation `json:"equation"`
}

type Usecase interface {
	CreateTask(ctx context.Context, in Input) (Output, error)
}
