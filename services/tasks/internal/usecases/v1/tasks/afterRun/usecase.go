package tasksAfterRun

import (
	"context"
	"fmt"
	"path/filepath"

	"github.com/google/uuid"

	"github.com/AridanWarlock/pinnAutomizer/pkg/errs"
	"github.com/AridanWarlock/pinnAutomizer/pkg/logger"
	"github.com/AridanWarlock/pinnAutomizer/pkg/postgres/poolx"
	"github.com/AridanWarlock/pinnAutomizer/tasks/internal/domain"
)

type Postgres interface {
	GetTaskByIDForUpdate(ctx context.Context, id uuid.UUID) (domain.Task, error)
	UpdateTask(ctx context.Context, task domain.Task) (domain.Task, error)

	poolx.TxManager
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
	task, err := u.postgres.GetTaskByIDForUpdate(ctx, in.ID)
	if err != nil {
		return fmt.Errorf("get task from postgres: %w", err)
	}

	changed, err := task.Complete(in.Error)
	if err != nil {
		return fmt.Errorf("%w: %v", errs.ErrConflict, err)
	}
	if !changed {
		return nil
	}

	plotPath := filepath.Join(task.OutputPath, "fig.pdf")
	if u.fileService.Exists(plotPath) {
		task.PlotPath = &plotPath
	}

	_, err = u.postgres.UpdateTask(ctx, task)
	if err != nil {
		return fmt.Errorf("update task: %w", err)
	}
	return nil
}
