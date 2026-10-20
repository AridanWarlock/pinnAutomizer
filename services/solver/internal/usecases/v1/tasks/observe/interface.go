package tasksObserve

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
	// Finished means this call committed the result and its event.
	Finished core.Changed
}

type Usecase interface {
	ObserveTask(context.Context, Input) (Output, error)
}
