package update_script_after_translate

import (
	"github.com/google/uuid"
	"pinnAutomizer/pkg/validate"
)

type Input struct {
	ID   uuid.UUID `validate:"required,uuid"`
	Text string
}

func (i Input) Validate() error {
	return validate.V.Struct(i)
}
