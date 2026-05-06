package tasksDelete

import (
	"context"
	"fmt"
	"github.com/AridanWarlock/pinnAutomizer/pkg/logger"
	"github.com/AridanWarlock/pinnAutomizer/pkg/postgres/poolx"
	"github.com/rs/zerolog"

	"github.com/AridanWarlock/pinnAutomizer/pkg/core"
	"github.com/AridanWarlock/pinnAutomizer/pkg/errs"
	"github.com/AridanWarlock/pinnAutomizer/tasks/internal/domain"
	"github.com/google/uuid"
)

type FileService interface {
	RemoveAll(dir string) error
}

type Postgres interface {
	GetTaskByIDAndUserID(ctx context.Context, id, userID uuid.UUID) (domain.Task, error)
	DeleteTaskByIDAndUserID(ctx context.Context, id uuid.UUID, userID uuid.UUID) error

	poolx.TxManager
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

func (u *usecase) DeleteTask(ctx context.Context, in Input) error {
	if err := in.Validate(); err != nil {
		return fmt.Errorf("%w: %v", errs.ErrInvalidArgument, err)
	}
	log := logger.FromContext(ctx)
	auth := core.MustAuthInfoFromContext(ctx)

	err := u.postgres.InTransaction(ctx, func(ctx context.Context) error {
		return u.deleteTaskIfAvailable(ctx, in, auth.UserID, log)
	})

	if err != nil {
		return fmt.Errorf("delete task transaction: %w", err)
	}

	return nil
}

func (u *usecase) deleteTaskIfAvailable(ctx context.Context, in Input, userID uuid.UUID, log zerolog.Logger) error {
	task, err := u.postgres.GetTaskByIDAndUserID(ctx, in.TaskID, userID)
	if err != nil {
		return fmt.Errorf("get task from postgres: %w", err)
	}

	if task.InQueue() || task.IsRunning() {
		return fmt.Errorf("%w: task is running", errs.ErrInvalidArgument)
	}

	err = u.postgres.DeleteTaskByIDAndUserID(ctx, task.ID, userID)
	if err != nil {
		return fmt.Errorf("delete task in postgres: %w", err)
	}

	err = u.fileService.RemoveAll(task.DataPath)
	if err != nil {
		log.Error().Err(err).Msgf("failed to delete task data dir: %s", task.DataPath)
	}
	err = u.fileService.RemoveAll(task.OutputPath)
	if err != nil {
		log.Error().Err(err).Msgf("failed to delete task output dir: %s", task.DataPath)
	}

	return nil
}
