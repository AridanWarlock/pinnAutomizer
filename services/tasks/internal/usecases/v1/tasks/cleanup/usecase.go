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
	GetFileCleanups(
		ctx context.Context,
		afterTaskID uuid.UUID,
		batch limit.Limit,
	) ([]domain.FileCleanup, error)
	DeleteFileCleanup(ctx context.Context, taskID uuid.UUID) error
}

type FileService interface {
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
	jobs, err := u.postgres.GetFileCleanups(ctx, in.AfterTaskID, u.batch)
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
		if err := u.cleanup(ctx, job); err != nil {
			failures = errors.Join(failures, fmt.Errorf("cleanup task %s: %w", job.TaskID, err))
		}
		out.NextTaskID = job.TaskID
	}
	return out, failures
}

func (u *usecase) cleanup(ctx context.Context, job domain.FileCleanup) error {
	dataErr := u.fileService.RemoveAll(job.DataPath)
	if err := ctx.Err(); err != nil {
		return errors.Join(dataErr, err)
	}
	outputErr := u.fileService.RemoveAll(job.OutputPath)
	if err := errors.Join(dataErr, outputErr); err != nil {
		return err
	}
	return u.postgres.DeleteFileCleanup(ctx, job.TaskID)
}
