package tasksRun

import (
	"context"

	"github.com/AridanWarlock/pinnAutomizer/pkg/validate"
	"github.com/google/uuid"
)

type Input struct {
	TaskID uuid.UUID `validate:"required,uuid"`
}

func (i Input) Validate() error {
	return validate.V.Struct(i)
}

type Usecase interface {
	RunTask(ctx context.Context, in Input) error
}
