package tasksPlot

import (
	"context"
	"errors"
	"fmt"
	"github.com/AridanWarlock/pinnAutomizer/pkg/core"
	"github.com/AridanWarlock/pinnAutomizer/pkg/errs"
	"github.com/AridanWarlock/pinnAutomizer/tasks/internal/domain"
	"github.com/rs/zerolog/log"
	"io"

	"github.com/google/uuid"
)

type Postgres interface {
	GetTaskByIDAndUserID(ctx context.Context, taskID, userID uuid.UUID) (domain.Task, error)
}

type FileWriter interface {
	Write(path string, writer io.Writer) error
}

type usecase struct {
	postgres   Postgres
	fileWriter FileWriter
}

func New(
	postgres Postgres,
	fileWriter FileWriter,
) Usecase {
	return &usecase{
		postgres:   postgres,
		fileWriter: fileWriter,
	}
}

func (u *usecase) DownloadTaskPlot(ctx context.Context, in Input) error {
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
	if task.PlotPath == nil {
		return fmt.Errorf("%w: task without plot", errs.ErrInvalidArgument)
	}

	if err := u.fileWriter.Write(*task.PlotPath, in.Writer); err != nil {
		if errors.Is(err, errs.ErrNotExists) {
			return fmt.Errorf("%w: plot file not found: %v", errs.ErrInvalidArgument, err)
		}

		return fmt.Errorf("write pdf plot file: %w", err)
	}

	return nil
}
