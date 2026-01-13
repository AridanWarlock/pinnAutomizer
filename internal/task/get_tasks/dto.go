package get_tasks

import (
	"github.com/google/uuid"
	"pinnAutomizer/internal/domain"
	"pinnAutomizer/pkg/validate"
)

type Input struct {
	IDs    []uuid.UUID `validate:"required,max=20,dive,required,uuid"`
	UserID uuid.UUID   `validate:"required,uuid"`
}

func (i Input) Validate() error {
	return validate.V.Struct(i)
}

type Output struct {
	TasksToEquation map[*domain.Task]domain.Equation
}
