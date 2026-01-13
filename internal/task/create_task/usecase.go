package create_task

import (
	"context"
	"github.com/rs/zerolog"
	"pinnAutomizer/internal/domain"
	"pinnAutomizer/pkg/tx"
)

type Postgres interface {
	CreateTask(ctx context.Context, task domain.Task) (domain.Task, error)
	GetEquationByType(ctx context.Context, equationType string) (domain.Equation, error)

	tx.Wrapper
}

type Kafka interface {
	TrainTask(ctx context.Context, task domain.Task) error
}

type Usecase struct {
	postgres Postgres
	kafka    Kafka

	log zerolog.Logger
}

var usecase *Usecase

func New(
	postgres Postgres,
	kafka Kafka,
	log zerolog.Logger,
) *Usecase {
	uc := &Usecase{
		postgres: postgres,
		kafka:    kafka,

		log: log.With().Str("component", "usecase: task.CreateTask").Logger(),
	}

	usecase = uc

	return uc
}

func (u *Usecase) CreateTask(ctx context.Context, in Input) (Output, error) {
	log := u.log.With().Ctx(ctx).Logger()

	if err := in.Validate(); err != nil {
		log.Info().Err(err).Msg("input validation error")
		return Output{}, err
	}

	equation, err := u.postgres.GetEquationByType(ctx, in.EquationType)

	if err != nil {
		log.Info().Err(err).Msg("getting equation from postgres error")
		return Output{}, err
	}

	task, err := domain.NewTask(
		in.Name,
		in.Description,
		domain.TaskStatusCreated,
		in.Constants,
		in.UserID,
		equation.ID,
	)
	if err != nil {
		log.Info().Err(err).Msg("creating task error")
		return Output{}, err
	}

	err = u.postgres.Wrap(ctx, func(ctx context.Context) error {
		task, err = u.postgres.CreateTask(ctx, task)
		if err != nil {
			log.Error().Err(err).Msg("saving task in postgres error")
			return err
		}

		if err := u.kafka.TrainTask(ctx, task); err != nil {
			log.Error().Err(err).Msg("saving task in kafka error")
			return err
		}
		return nil
	})

	if err != nil {
		return Output{}, err
	}

	return Output{
		Task:     task,
		Equation: equation,
	}, err
}
