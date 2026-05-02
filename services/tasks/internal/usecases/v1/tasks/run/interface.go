package tasksRun

import (
	"context"

	"github.com/AridanWarlock/pinnAutomizer/pkg/validate"

	"github.com/google/uuid"
)

type Input struct {
	TaskID uuid.UUID `validate:"required,uuid"`

	IdempotencyKey string `validate:"required"`
}

func (i Input) Validate() error {
	return validate.V.Struct(i)
}

type Usecase interface {
	RunTask(cxt context.Context, in Input) error
}
