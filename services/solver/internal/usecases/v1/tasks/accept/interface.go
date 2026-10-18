package tasksAccept

import (
	"context"
	"fmt"

	"github.com/google/uuid"

	"github.com/AridanWarlock/pinnAutomizer/pkg/core"
	"github.com/AridanWarlock/pinnAutomizer/pkg/errs"
	"github.com/AridanWarlock/pinnAutomizer/pkg/validate"
	"github.com/AridanWarlock/pinnAutomizer/solver/internal/domain"
)

type Input struct {
	RunID  uuid.UUID `validate:"required,uuid"`
	TaskID uuid.UUID `validate:"required,uuid"`
	Mode   string    `validate:"required,oneof=train retrain predict"`
}

func (in Input) Validate() error {
	if err := validate.V.Struct(in); err != nil {
		return fmt.Errorf("%w: %v", errs.ErrInvalidArgument, err)
	}
	return nil
}

type Output struct {
	Attempt domain.Attempt
	Created core.Created
}

type Usecase interface {
	AcceptTask(context.Context, Input) (Output, error)
}
