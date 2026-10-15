package tasksOnRun

import (
	"context"

	"github.com/google/uuid"

	"github.com/AridanWarlock/pinnAutomizer/pkg/validate"
)

type Input struct {
	ID uuid.UUID `validate:"required,uuid"`
}

func (i Input) Validate() error {
	return validate.V.Struct(i)
}

type Usecase interface {
	UpdateTaskOnRun(ctx context.Context, in Input) error
}
