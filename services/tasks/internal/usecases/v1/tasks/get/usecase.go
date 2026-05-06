package tasksGet

import (
	"context"
	"fmt"

	"github.com/AridanWarlock/pinnAutomizer/pkg/core"
	"github.com/AridanWarlock/pinnAutomizer/pkg/core/pagination"
	"github.com/AridanWarlock/pinnAutomizer/pkg/errs"
	"github.com/AridanWarlock/pinnAutomizer/tasks/internal/domain"
	"github.com/rs/zerolog/log"

	"github.com/google/uuid"
)

type Postgres interface {
	GetTasksByUserID(ctx context.Context, userID uuid.UUID, opts *pagination.Options) ([]domain.Task, error)
	GetTasksCount(ctx context.Context, userID uuid.UUID) (int, error)
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

func (u *usecase) GetTasks(ctx context.Context, in Input) (Output, error) {
	if err := in.Validate(); err != nil {
		return Output{}, fmt.Errorf("%w: %v", errs.ErrInvalidArgument, err)
	}
	auth := core.MustAuthInfoFromContext(ctx)

	tasks, err := u.postgres.GetTasksByUserID(ctx, auth.UserID, in.Pagination)
	if err != nil {
		log.Error().Err(err).Msg("usecase: postgres.GetTasksByIDs")
		return Output{}, fmt.Errorf("getting tasks by id from postgres: %w", err)
	}

	total, err := u.postgres.GetTasksCount(ctx, auth.UserID)
	if err != nil {
		log.Error().Err(err).Msg("usecase: postgres.GetTasksCount")
		return Output{}, fmt.Errorf("getting tasks count: %w", err)
	}

	return Output{
		Tasks: tasks,
		Total: total,
	}, nil
}
