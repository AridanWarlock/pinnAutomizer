package get_tasks

import (
	"pinnAutomizer/internal/domain"
	"pinnAutomizer/pkg/validate"

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
	TasksToEquation map[*domain.Task]domain.Equation
}
