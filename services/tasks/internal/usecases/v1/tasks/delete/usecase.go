package tasksDelete

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"

	"github.com/AridanWarlock/pinnAutomizer/pkg/core"
	"github.com/AridanWarlock/pinnAutomizer/pkg/errs"
	"github.com/AridanWarlock/pinnAutomizer/pkg/postgres/poolx"
	"github.com/AridanWarlock/pinnAutomizer/tasks/internal/domain"
)

type Postgres interface {
	GetTaskByIDAndUserIDForUpdate(ctx context.Context, id, userID uuid.UUID) (domain.Task, error)
	DeleteTaskByIDAndUserID(ctx context.Context, id uuid.UUID, userID uuid.UUID) error
	QueueFileCleanupJob(ctx context.Context, job domain.FileCleanupJob) error

	poolx.TxManager
}

type usecase struct {
	postgres Postgres
}

func New(postgres Postgres) Usecase {
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

func (u *usecase) deleteTaskIfAvailable(
	ctx context.Context,
	in Input,
	userID uuid.UUID,
) error {
	task, err := u.postgres.GetTaskByIDAndUserIDForUpdate(ctx, in.TaskID, userID)
	if errors.Is(err, errs.ErrNotFound) {
		return nil
	}
	if err != nil {
		return fmt.Errorf("get task from postgres: %w", err)
	}

	if !task.CanDelete() {
		return fmt.Errorf("%w: task is in progress", errs.ErrConflict)
	}

	job, err := domain.NewFileCleanupJob(task.ID, task.DataPath, task.OutputPath)
	if err != nil {
		return fmt.Errorf("plan file cleanup: %w", err)
	}
	if err := u.postgres.QueueFileCleanupJob(ctx, job); err != nil {
		return fmt.Errorf("queue file cleanup: %w", err)
	}
	if err := u.postgres.DeleteTaskByIDAndUserID(ctx, task.ID, userID); err != nil {
		return fmt.Errorf("delete task in postgres: %w", err)
	}
	return nil
}
