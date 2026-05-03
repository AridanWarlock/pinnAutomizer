package tasksResults

import (
	"context"
	"fmt"
	"io"

	"github.com/AridanWarlock/pinnAutomizer/pkg/core"
	"github.com/AridanWarlock/pinnAutomizer/pkg/errs"
	"github.com/AridanWarlock/pinnAutomizer/tasks/internal/domain"
	"github.com/rs/zerolog/log"

	"github.com/google/uuid"
)

type Postgres interface {
	GetTaskByIDAndUserID(ctx context.Context, taskID, userID uuid.UUID) (domain.Task, error)
}

type Zipper interface {
	ZipFiles(dir string, writer io.Writer) error
}

type usecase struct {
	postgres Postgres
	zipper   Zipper
}

func New(
	postgres Postgres,
	zipper Zipper,
) Usecase {
	return &usecase{
		postgres: postgres,
		zipper:   zipper,
	}
}

func (u *usecase) DownloadTaskResults(ctx context.Context, in Input) error {
	if err := in.Validate(); err != nil {
		return fmt.Errorf("%w: %v", errs.ErrInvalidArgument, err)
	}
	auth := core.MustAuthInfoFromContext(ctx)

	task, err := u.postgres.GetTaskByIDAndUserID(ctx, in.TaskID, auth.UserID)
	if err != nil {
		log.Error().Err(err).Msg("usecase: postgres.GetTasksByIDs")
		return fmt.Errorf("getting tasks by id from postgres: %w", err)
	}

	if !task.IsEnded() {
		return fmt.Errorf("%w: task in progress", errs.ErrInvalidArgument)
	}

	if err := u.zipper.ZipFiles(task.OutputPath, in.Writer); err != nil {
		return fmt.Errorf("zip output files: %w", err)
	}

	return nil
}
