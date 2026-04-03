package create_task

import (
	"pinnAutomizer/internal/domain"
	"pinnAutomizer/pkg/validate"

	"github.com/google/uuid"
)

type Input struct {
	Name         string `validate:"required"`
	Description  string
	Constants    map[string]any
	UserID       uuid.UUID `validate:"required,uuid"`
	EquationType string    `validate:"required,oneof=heat wave"`

	IdempotencyKey string `validate:"required"`
}

func (i Input) Validate() error {
	return validate.V.Struct(i)
}

type Output struct {
	Task     domain.Task     `json:"task"`
	Equation domain.Equation `json:"equation"`
}
