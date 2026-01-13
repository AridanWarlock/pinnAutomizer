package solve_task

import (
	"context"
	"pinnAutomizer/internal/domain"

	"github.com/google/uuid"
	"github.com/rs/zerolog"
)

type Postgres interface {
	GetTaskByID(ctx context.Context, id uuid.UUID) (domain.Task, error)
}

type Kafka interface {
	PublishTaskToSolve(ctx context.Context, task domain.Task) error
}

type Usecase struct {
	postgres Postgres
	kafka    Kafka

	log zerolog.Logger
}

var usecase *Usecase

func New(postgres Postgres, kafka Kafka, log zerolog.Logger) *Usecase {
	usecase = &Usecase{
		postgres: postgres,
		kafka:    kafka,

		log: log.With().Str("component", "usecase: task.SolveTask").Logger(),
	}

	return usecase
}

func (u *Usecase) SolveTask(ctx context.Context, in Input) error {
	log := u.log.With().Ctx(ctx).Logger()

	if err := in.Validate(); err != nil {
		log.Error().Err(err).Msg("usecase: input.Validate")
		return err
	}

	task, err := u.postgres.GetTaskByID(ctx, in.TaskID)
	if err != nil {
		log.Error().Err(err).Msg("usecase: postgres.GetTaskByID")
		return err
	}

	if task.UserID != in.UserID {
		log.Error().Err(domain.ErrIncorrectUser).Msg("usecase: IncorrectUser")
		return domain.ErrIncorrectUser
	}

	err = u.kafka.PublishTaskToSolve(ctx, task)
	if err != nil {
		log.Error().Err(err).Msg("usecase: kafka.PublishTaskToSolve")
		return err
	}

	return nil
}
