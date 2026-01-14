package create_task

import (
	"context"
	"encoding/json"
	"pinnAutomizer/internal/domain"
	"pinnAutomizer/pkg/tx"

	"github.com/google/uuid"
	"github.com/rs/zerolog"
)

type Postgres interface {
	CreateTask(ctx context.Context, task domain.Task) (domain.Task, error)
	GetEquationByType(ctx context.Context, equationType string) (domain.Equation, error)
	UpdateTaskStatusByID(ctx context.Context, id uuid.UUID, status string) error
	PublishEvent(ctx context.Context, event domain.Event) error

	tx.Wrapper
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

		log: log.With().Str("component", "usecase: task.CreateTask").Logger(),
	}

	return usecase
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
			log.Error().Err(err).Msg("usecase: postgres.CreateTask")
			return err
		}

		event, err := u.createTaskTrainEvent(task)
		if err != nil {
			log.Error().Err(err).Msg("usecase: createTaskTrainEvent")
			return err
		}

		err = u.postgres.PublishEvent(ctx, event)
		if err != nil {
			log.Error().Err(err).Msg("usecase: postgres.PublishEvent")
			return err
		}

		return nil
	})

	if err != nil {
		return domain.Task{}, err
	}
	return task, nil
}

func (u *Usecase) createTaskTrainEvent(task domain.Task) (domain.Event, error) {
	msg := domain.TrainMessage{
		TaskID:      task.ID,
		MatFilePath: task.TrainingDataPath,
		Constants:   task.Constants,
	}

	jsonMsg, err := json.Marshal(msg)
	if err != nil {
		return domain.Event{}, err
	}

	return domain.NewEvent("to-train", jsonMsg), nil
}
