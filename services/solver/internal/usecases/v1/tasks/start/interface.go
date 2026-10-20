package tasksStart

import (
	"context"
	"fmt"

	"github.com/google/uuid"

	"github.com/AridanWarlock/pinnAutomizer/pkg/core"
	"github.com/AridanWarlock/pinnAutomizer/pkg/errs"
	"github.com/AridanWarlock/pinnAutomizer/pkg/validate"
)

type Input struct {
	RunID uuid.UUID `validate:"required,uuid"`
}

func (in Input) Validate() error {
	if err := validate.V.Struct(in); err != nil {
		return fmt.Errorf("%w: %v", errs.ErrInvalidArgument, err)
	}
	return nil
}

type Output struct {
	// StartRequested means this call consumed the durable permission to send
	// Start, not that the container is running or the computation has succeeded.
	StartRequested core.Changed
}

type Usecase interface {
	StartTask(context.Context, Input) (Output, error)
}
