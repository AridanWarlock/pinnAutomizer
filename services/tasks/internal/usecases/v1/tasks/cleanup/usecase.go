package tasksCleanup

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"

	"github.com/AridanWarlock/pinnAutomizer/pkg/core/limit"
	"github.com/AridanWarlock/pinnAutomizer/tasks/internal/domain"
)

type Postgres interface {
	GetFileCleanupJobs(
		ctx context.Context,
		afterTaskID uuid.UUID,
		batch limit.Limit,
	) ([]domain.FileCleanupJob, error)
	DeleteFileCleanupJob(ctx context.Context, taskID uuid.UUID) error
	HasFileCleanupJob(ctx context.Context, taskID uuid.UUID) (bool, error)
}

type FileService interface {
	WithDirectoryLock(
		ctx context.Context,
		dataPath string,
		process func(context.Context) error,
	) error
	RemoveAll(dir string) error
}

type usecase struct {
	postgres    Postgres
	fileService FileService
	batch       limit.Limit
}

func New(
	postgres Postgres,
	fileService FileService,
	batch limit.Limit,
) Usecase {
	return &usecase{
		postgres:    postgres,
		fileService: fileService,
		batch:       batch,
	}
}

func (u *usecase) CleanupPending(ctx context.Context, in Input) (Output, error) {
	out := Output{NextTaskID: in.AfterTaskID}
	if err := ctx.Err(); err != nil {
		return out, err
	}
	jobs, err := u.postgres.GetFileCleanupJobs(ctx, in.AfterTaskID, u.batch)
	if err != nil {
		return out, fmt.Errorf("list file cleanup: %w", err)
	}
	if len(jobs) == 0 {
		return Output{}, nil
	}
	var failures error
	for _, job := range jobs {
		if err := ctx.Err(); err != nil {
			return out, errors.Join(failures, err)
		}
		if err := u.fileService.WithDirectoryLock(ctx, job.DataPath, func(ctx context.Context) error {
			pending, err := u.postgres.HasFileCleanupJob(ctx, job.TaskID)
			if err != nil {
				return err
			}
			if !pending {
				return nil
			}
			return u.cleanup(ctx, job)
		}); err != nil {
			failures = errors.Join(failures, fmt.Errorf("cleanup task %s: %w", job.TaskID, err))
		}
		out.NextTaskID = job.TaskID
	}
	return out, failures
}

func (u *usecase) cleanup(ctx context.Context, job domain.FileCleanupJob) error {
	dataErr := u.fileService.RemoveAll(job.DataPath)
	if err := ctx.Err(); err != nil {
		return errors.Join(dataErr, err)
	}
	outputErr := u.fileService.RemoveAll(job.OutputPath)
	if err := errors.Join(dataErr, outputErr); err != nil {
		return err
	}
	return u.postgres.DeleteFileCleanupJob(ctx, job.TaskID)
}
