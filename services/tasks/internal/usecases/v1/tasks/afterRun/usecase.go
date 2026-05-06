package tasksAfterRun

import (
	"context"
	"fmt"
	"path/filepath"

	"github.com/AridanWarlock/pinnAutomizer/pkg/errs"
	"github.com/AridanWarlock/pinnAutomizer/pkg/logger"
	"github.com/AridanWarlock/pinnAutomizer/tasks/internal/domain"
	"github.com/google/uuid"
)

type Postgres interface {
	GetTaskByID(ctx context.Context, id uuid.UUID) (domain.Task, error)
	UpdateTask(ctx context.Context, task domain.Task) (domain.Task, error)

	InTransaction(ctx context.Context, inTx func(ctx context.Context) error) error
}

type FileService interface {
	Exists(filepath string) bool
}

type usecase struct {
	postgres    Postgres
	fileService FileService
}

func New(
	postgres Postgres,
	fileService FileService,
) Usecase {
	return &usecase{
		postgres:    postgres,
		fileService: fileService,
	}
}

func (u *usecase) UpdateTaskAfterRun(ctx context.Context, in Input) error {
	log := logger.FromContext(ctx)

	if err := in.Validate(); err != nil {
		return fmt.Errorf("%w: %v", errs.ErrInvalidArgument, err)
	}

	log.Info().Msg("updating task after train")

	err := u.postgres.InTransaction(ctx, func(ctx context.Context) error {
		return u.updateTaskAfterRun(ctx, in)
	})

	if err != nil {
		return fmt.Errorf("update task after train transaction: %w", err)
	}

	return nil
}

func (u *usecase) updateTaskAfterRun(ctx context.Context, in Input) error {
	task, err := u.postgres.GetTaskByID(ctx, in.ID)
	if err != nil {
		return fmt.Errorf("get task from postgres: %w", err)
	}

	plotPath := filepath.Join(task.OutputPath, "fig.pdf")
	if u.fileService.Exists(plotPath) {
		task.PlotPath = &plotPath
	}

	if in.Error != nil {
		task.Status = domain.TaskStatusError
		task.Error = in.Error
	} else {
		task.Status = domain.TaskStatusDone
	}

	_, err = u.postgres.UpdateTask(ctx, task)
	if err != nil {
		return fmt.Errorf("update task: %w", err)
	}
	return nil
}
