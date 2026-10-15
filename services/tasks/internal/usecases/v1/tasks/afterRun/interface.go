package tasksAfterRun

import (
	"context"

	"github.com/google/uuid"

	"github.com/AridanWarlock/pinnAutomizer/pkg/validate"
)

type Input struct {
	ID    uuid.UUID `validate:"required,uuid"`
	Error *string
}

func (i Input) Validate() error {
	return validate.V.Struct(i)
}

type Usecase interface {
	UpdateTaskAfterRun(ctx context.Context, in Input) error
}
