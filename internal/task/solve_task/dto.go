package solve_task

import (
	"github.com/google/uuid"
	"pinnAutomizer/pkg/validate"
)

type Input struct {
	TaskID uuid.UUID `validate:"required,uuid"`
	UserID uuid.UUID `validate:"required,uuid"`

	Constants map[string]any
}

func (i Input) Validate() error {
	return validate.V.Struct(i)
}
