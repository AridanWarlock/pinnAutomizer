package tasksCreate

import (
	"context"
	"fmt"
	"os"

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
	WithTaskFiles(
		ctx context.Context,
		dataPath string,
		process func(context.Context) error,
	) error
	MkdirAll(name string, perm os.FileMode) error
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

	job, err := domain.NewFileCleanup(task.ID, task.DataPath, task.OutputPath)
	if err != nil {
		return Output{}, fmt.Errorf("plan creation cleanup: %w", err)
	}
	err = u.fileStore.WithTaskFiles(ctx, task.DataPath, func(ctx context.Context) error {
		if err := u.postgres.QueueFileCleanup(ctx, job); err != nil {
			return fmt.Errorf("reserve creation cleanup: %w", err)
		}
		if err := u.storeTaskFiles(ctx, task, in); err != nil {
			return err
		}
		// If commit is ambiguous, keep the durable outcome in PostgreSQL; do not
		// compensate by deleting files here, since the task may have committed.
		return u.postgres.InTransaction(ctx, func(ctx context.Context) error {
			created, err := u.postgres.CreateTask(ctx, task)
			if err != nil {
				return fmt.Errorf("create task in postgres: %w", err)
			}
			if err := u.postgres.DeleteFileCleanup(ctx, task.ID); err != nil {
				return fmt.Errorf("confirm creation: %w", err)
			}
			task = created
			return nil
		})
	})
	if err != nil {
		return Output{}, fmt.Errorf("create task: %w", err)
	}

	return Output{Task: task}, nil
}

func (u *usecase) storeTaskFiles(
	ctx context.Context,
	task domain.Task,
	in Input,
) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if err := u.fileStore.MkdirAll(task.DataPath, 0755); err != nil {
		return fmt.Errorf("create task data dir: %w", err)
	}
	if err := ctx.Err(); err != nil {
		return err
	}
	if err := u.fileStore.MkdirAll(task.OutputPath, 0755); err != nil {
		return fmt.Errorf("create task out dir: %w", err)
	}
	if err := ctx.Err(); err != nil {
		return err
	}
	if err := u.fileStore.Store(task.DataPath, in.Files); err != nil {
		return fmt.Errorf("store files: %w", err)
	}
	return ctx.Err()
}
