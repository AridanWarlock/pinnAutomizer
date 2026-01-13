package update_task_status_on_train

import (
	"context"
	"github.com/google/uuid"
	"github.com/rs/zerolog"
	"pinnAutomizer/internal/domain"
)

type Postgres interface {
	UpdateTaskStatusByID(ctx context.Context, id uuid.UUID, status string) error
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
	usecase = &Usecase{
		postgres: postgres,

		log: log.With().Str("component", "usecase: task.UpdateTaskStatusOnTrain").Logger(),
	}

	return usecase
}

func (u *Usecase) UpdateTaskStatusOnTrain(ctx context.Context, in Input) error {
	log := u.log.With().Ctx(ctx).Logger()

	if err := in.Validate(); err != nil {
		log.Error().Err(err).Msg("input validation failed")
		return err
	}

	err := u.postgres.UpdateTaskStatusByID(ctx, in.ID, string(domain.TaskStatusTraining))
	if err != nil {
		log.Error().Err(err).Msg("postgres updating task status error")
		return err
	}

	return nil
}
