package create_task

import (
	"context"
	"pinnAutomizer/internal/domain"
	"pinnAutomizer/pkg/tx"

	"github.com/google/uuid"
	"github.com/rs/zerolog"
)

type Postgres interface {
	CreateTask(ctx context.Context, task domain.Task) (domain.Task, error)
	GetEquationByType(ctx context.Context, equationType string) (domain.Equation, error)
	UpdateTaskStatusByID(ctx context.Context, id uuid.UUID, status string) error

	tx.Wrapper
}

type Kafka interface {
	PublishTaskToTrain(ctx context.Context, task domain.Task) error
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

	task, err = u.createAndPublishTask(ctx, task)

	if err != nil {
		return Output{}, err
	}

	return Output{
		Task:     task,
		Equation: equation,
	}, err
}

func (u *Usecase) createAndPublishTask(ctx context.Context, task domain.Task) (domain.Task, error) {
	log := u.log.With().Ctx(ctx).Logger()

	err := u.postgres.Wrap(ctx, func(ctx context.Context) error {
		var err error

		task, err = u.postgres.CreateTask(ctx, task)
		if err != nil {
			log.Error().Err(err).Msg("saving task in postgres error")
			return err
		}

		err = u.kafka.PublishTaskToTrain(ctx, task)
		if err != nil {
			log.Error().Err(err).Msg("saving task in kafka_produce error")
			return err
		}

		return nil
	})

	if err != nil {
		return domain.Task{}, err
	}
	return task, nil
}
