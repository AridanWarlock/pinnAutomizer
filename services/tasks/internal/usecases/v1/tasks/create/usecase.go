package tasksCreate

import (
	"context"
	"fmt"

	"github.com/google/uuid"

	"github.com/AridanWarlock/pinnAutomizer/pkg/core"
	"github.com/AridanWarlock/pinnAutomizer/pkg/errs"
	"github.com/AridanWarlock/pinnAutomizer/pkg/postgres/poolx"
	"github.com/AridanWarlock/pinnAutomizer/tasks/internal/domain"
)

type Postgres interface {
	CreateTask(ctx context.Context, task domain.Task) (domain.Task, error)
	QueueFileCleanup(ctx context.Context, job domain.FileCleanup) error
	DeleteFileCleanup(ctx context.Context, taskID uuid.UUID) error
	poolx.TxManager
}

type TaskFileStore interface {
	WithDirectoryLock(
		ctx context.Context,
		dataPath string,
		process func(context.Context) error,
	) error
	MkdirAll(name string) error
	Store(dir string, files []domain.TaskFile) error
}

type usecase struct {
	postgres  Postgres
	fileStore TaskFileStore
}

func New(
	postgres Postgres,
	fileStore TaskFileStore,
) Usecase {
	return &usecase{
		postgres:  postgres,
		fileStore: fileStore,
	}
}

func (u *usecase) CreateTask(ctx context.Context, in Input) (Output, error) {
	authInfo := core.MustAuthInfoFromContext(ctx)

	if err := in.Validate(); err != nil {
		return Output{}, fmt.Errorf("%w: %v", errs.ErrInvalidArgument, err)
	}

	task, err := domain.NewTask(
		in.Name,
		authInfo.UserID,
		in.Mode,
		in.Description,
	)
	if err != nil {
		return Output{}, fmt.Errorf("create task model: %w", err)
	}

	// Acquire filesystem ownership before publishing the compensation job.
	err = u.fileStore.WithDirectoryLock(ctx, task.DataPath, func(ctx context.Context) error {
		if err := u.prepareTaskFiles(ctx, task, in); err != nil {
			return err
		}
		created, err := u.confirmTask(ctx, task)
		if err == nil {
			task = created
		}
		return err
	})
	if err != nil {
		return Output{}, fmt.Errorf("create task: %w", err)
	}

	return Output{Task: task}, nil
}

func (u *usecase) prepareTaskFiles(
	ctx context.Context,
	task domain.Task,
	in Input,
) error {
	job, err := domain.NewFileCleanup(task.ID, task.DataPath, task.OutputPath)
	if err != nil {
		return fmt.Errorf("plan creation cleanup: %w", err)
	}
	if err := u.postgres.QueueFileCleanup(ctx, job); err != nil {
		return fmt.Errorf("reserve creation cleanup: %w", err)
	}
	return u.storeTaskFiles(task, in)
}

func (u *usecase) storeTaskFiles(task domain.Task, in Input) error {
	if err := u.fileStore.MkdirAll(task.DataPath); err != nil {
		return fmt.Errorf("create task data dir: %w", err)
	}
	if err := u.fileStore.MkdirAll(task.OutputPath); err != nil {
		return fmt.Errorf("create task out dir: %w", err)
	}
	if err := u.fileStore.Store(task.DataPath, in.Files); err != nil {
		return fmt.Errorf("store files: %w", err)
	}
	return nil
}

// Ownership is held until this transaction returns. An ambiguous commit must
// be reconciled by the worker, never by synchronously removing the files here.
func (u *usecase) confirmTask(
	ctx context.Context,
	task domain.Task,
) (domain.Task, error) {
	var created domain.Task
	err := u.postgres.InTransaction(ctx, func(ctx context.Context) error {
		var err error
		created, err = u.postgres.CreateTask(ctx, task)
		if err != nil {
			return fmt.Errorf("create task in postgres: %w", err)
		}
		if err := u.postgres.DeleteFileCleanup(ctx, task.ID); err != nil {
			return fmt.Errorf("confirm creation: %w", err)
		}
		return nil
	})
	return created, err
}
