package tasksGet

import (
	"context"
	"fmt"
	"maps"
	"slices"

	"github.com/AridanWarlock/pinnAutomizer/pinn/internal/domain"
	"github.com/AridanWarlock/pinnAutomizer/pkg/errs"
	"github.com/rs/zerolog/log"

	"github.com/google/uuid"
)

type Postgres interface {
	GetTasksByIDs(ctx context.Context, ids []uuid.UUID, userID uuid.UUID) ([]domain.Task, error)
	GetEquationsByIDs(ctx context.Context, ids []uuid.UUID) ([]domain.Equation, error)
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

	tasks, err := u.postgres.GetTasksByIDs(ctx, in.IDs, in.UserID)
	if err != nil {
		log.Error().Err(err).Msg("usecase: postgres.GetTasksByIDs")
		return Output{}, fmt.Errorf("getting tasks by id from postgres: %w", err)
	}

	equationIDs := make(map[uuid.UUID]struct{})
	for _, task := range tasks {
		equationIDs[task.EquationID] = struct{}{}
	}

	equations, err := u.postgres.GetEquationsByIDs(ctx, slices.Collect(maps.Keys(equationIDs)))
	if err != nil {
		return Output{}, fmt.Errorf("getting equations from postgres: %w", err)
	}

	equationIDsToEquation := make(map[uuid.UUID]domain.Equation, len(equations))
	for _, equation := range equations {
		equationIDsToEquation[equation.ID] = equation
	}

	taskToEquations := make(map[*domain.Task]domain.Equation, len(tasks))
	for _, task := range tasks {
		taskToEquations[&task] = equationIDsToEquation[task.EquationID]
	}

	return Output{TasksToEquation: taskToEquations}, nil
}
