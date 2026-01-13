package update_task_status_after_train

import (
	"pinnAutomizer/pkg/validate"

	"github.com/google/uuid"
)

type Input struct {
	ID uuid.UUID `validate:"required,uuid"`
}

func (i Input) Validate() error {
	return validate.V.Struct(i)
}
