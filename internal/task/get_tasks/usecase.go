package get_tasks

import (
	"context"
	"github.com/google/uuid"
	"github.com/rs/zerolog"
	"maps"
	"pinnAutomizer/internal/domain"
	"slices"
)

type Postgres interface {
	GetTasksByIDs(ctx context.Context, ids []uuid.UUID, userID uuid.UUID) ([]domain.Task, error)
	GetEquationsByIDs(ctx context.Context, ids []uuid.UUID) ([]domain.Equation, error)
}

type Usecase struct {
	postgres Postgres

	log zerolog.Logger
}

var usecase *Usecase

func New(
	postgres Postgres,
	log zerolog.Logger,
) *Usecase {
	uc := &Usecase{
		postgres: postgres,

		log: log.With().Str("component", "task.GetTasks").Logger(),
	}

	usecase = uc

	return uc
}

func (u *Usecase) GetTasks(ctx context.Context, in Input) (Output, error) {
	log := u.log.With().Ctx(ctx).Logger()

	if err := in.Validate(); err != nil {
		log.Info().Err(err).Msg("input validation error")
		return Output{}, err
	}

	tasks, err := u.postgres.GetTasksByIDs(ctx, in.IDs, in.UserID)
	if err != nil {
		log.Error().Err(err).Msg("postgres getting tasks error")
		return Output{}, err
	}

	equationIDs := make(map[uuid.UUID]struct{})
	for _, task := range tasks {
		equationIDs[task.EquationID] = struct{}{}
	}

	equations, err := u.postgres.GetEquationsByIDs(ctx, slices.Collect(maps.Keys(equationIDs)))
	if err != nil {
		log.Error().Err(err).Msg("postgres getting equations error")
		return Output{}, err
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
