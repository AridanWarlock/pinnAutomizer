package tasksRun

import (
	"context"
	"fmt"

	"github.com/AridanWarlock/pinnAutomizer/pkg/errs"
	"github.com/AridanWarlock/pinnAutomizer/solver/internal/domain"
)

type PinnRunner interface {
	Run(ctx context.Context, task domain.MlTask) (int, error)
}

type usecase struct {
	runner PinnRunner
}

func New(runner PinnRunner) *usecase {
	return &usecase{
		runner: runner,
	}
}

func (u *usecase) RunTask(ctx context.Context, in Input) error {
	if err := in.Validate(); err != nil {
		return errs.ErrInvalidArgument
	}

	task, err := domain.NewMlTask(
		in.TaskID,
		domain.MlTaskModeTrain,
		nil,
	)
	if err != nil {
		return fmt.Errorf("%w: %v", errs.ErrInvalidArgument, err)
	}

	code, err := u.runner.Run(ctx, task)
	if err != nil {
		return fmt.Errorf("run pinn (code=%d): %w", code, err)
	}

	return nil
}
