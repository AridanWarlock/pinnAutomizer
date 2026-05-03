package tasksDelete

import (
	"context"
	"fmt"

	"github.com/AridanWarlock/pinnAutomizer/pkg/core"
	"github.com/AridanWarlock/pinnAutomizer/pkg/errs"
	"github.com/AridanWarlock/pinnAutomizer/tasks/internal/domain"
	"github.com/google/uuid"
)

type Postgres interface {
	GetTaskByIDAndUserID(ctx context.Context, id, userID uuid.UUID) (domain.Task, error)
	DeleteTaskByIDAndUserID(ctx context.Context, id uuid.UUID, userID uuid.UUID) error

	InTransaction(ctx context.Context, inTx func(ctx context.Context) error) error
}

type usecase struct {
	postgres Postgres
}

func New(
	postgres Postgres,
) Usecase {
	return &usecase{
		postgres: postgres,
	}
}

func (u *usecase) DeleteTask(ctx context.Context, in Input) error {
	if err := in.Validate(); err != nil {
		return fmt.Errorf("%w: %v", errs.ErrInvalidArgument, err)
	}
	auth := core.MustAuthInfoFromContext(ctx)

	err := u.postgres.InTransaction(ctx, func(ctx context.Context) error {
		return u.deleteTaskIfAvailable(ctx, in, auth.UserID)
	})

	if err != nil {
		return fmt.Errorf("delete task transaction: %w", err)
	}

	return nil
}

func (u *usecase) deleteTaskIfAvailable(ctx context.Context, in Input, userID uuid.UUID) error {
	task, err := u.postgres.GetTaskByIDAndUserID(ctx, in.TaskID, userID)
	if err != nil {
		return fmt.Errorf("get task from postgres: %w", err)
	}

	if task.IsRunning() {
		return fmt.Errorf("%w: task is running", errs.ErrInvalidArgument)
	}

	err = u.postgres.DeleteTaskByIDAndUserID(ctx, task.ID, userID)
	if err != nil {
		return fmt.Errorf("delete task in postgres: %w", err)
	}
	return nil
}
