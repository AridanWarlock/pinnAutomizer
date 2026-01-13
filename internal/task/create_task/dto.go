package create_task

import (
	"github.com/google/uuid"
	"pinnAutomizer/internal/domain"
	"pinnAutomizer/pkg/validate"
)

type Input struct {
	Name         string `validate:"required"`
	Description  string
	Constants    map[string]any
	UserID       uuid.UUID `validate:"required,uuid"`
	EquationType string    `validate:"required"`
}

func (i Input) Validate() error {
	return validate.V.Struct(i)
}

type Output struct {
	Task     domain.Task
	Equation domain.Equation
}
