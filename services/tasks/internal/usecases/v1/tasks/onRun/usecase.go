package tasksOnRun

import (
	"context"
	"fmt"
	"github.com/AridanWarlock/pinnAutomizer/pkg/postgres/poolx"

	"github.com/AridanWarlock/pinnAutomizer/pkg/errs"
	"github.com/AridanWarlock/pinnAutomizer/pkg/logger"
	"github.com/AridanWarlock/pinnAutomizer/tasks/internal/domain"
	"github.com/google/uuid"
)

type Postgres interface {
	GetTaskByIDForUpdate(ctx context.Context, id uuid.UUID) (domain.Task, error)
	UpdateTaskStatusByID(ctx context.Context, id uuid.UUID, newStatus, oldStatus domain.TaskStatus) error

	poolx.TxManager
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

func (u *usecase) UpdateTaskOnRun(ctx context.Context, in Input) error {
	log := logger.FromContext(ctx)

	if err := in.Validate(); err != nil {
		return fmt.Errorf("%w: %v", errs.ErrInvalidArgument, err)
	}

	log.Info().Msg("updating task on run")

	err := u.postgres.InTransaction(ctx, func(ctx context.Context) error {
		return u.updateTaskStatusIfNeeded(ctx, in)
	})

	if err != nil {
		return err
	}
	return nil
}

func (u *usecase) updateTaskStatusIfNeeded(ctx context.Context, in Input) error {
	task, err := u.postgres.GetTaskByIDForUpdate(ctx, in.ID)
	if err != nil {
		return fmt.Errorf("get task from postgres: %w", err)
	}

	changed, err := task.MarkRunning()
	if err != nil {
		return fmt.Errorf("%w: %v", errs.ErrConflict, err)
	}
	if !changed {
		return nil
	}

	err = u.postgres.UpdateTaskStatusByID(ctx, in.ID, domain.TaskStatusRunning, domain.TaskStatusInQueue)
	if err != nil {
		return fmt.Errorf("update task on run status in postgres: %w", err)
	}
	return nil
}
