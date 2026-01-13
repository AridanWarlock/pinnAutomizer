package solve_task

import (
	"pinnAutomizer/pkg/validate"

	"github.com/google/uuid"
)

type Input struct {
	TaskID uuid.UUID `validate:"required,uuid"`
	UserID uuid.UUID `validate:"required,uuid"`

	Constants map[string]any
}

func (i Input) Validate() error {
	return validate.V.Struct(i)
}
