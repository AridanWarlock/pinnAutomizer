package create_task

import (
	"context"
	"encoding/json"
	"pinnAutomizer/internal/domain"
	"pinnAutomizer/pkg/tx"
	"time"

	"github.com/rs/zerolog"
)

type Postgres interface {
	CreateTask(ctx context.Context, task domain.Task) (domain.Task, error)
	GetEquationByType(ctx context.Context, equationType string) (domain.Equation, error)
	PublishEvent(ctx context.Context, event domain.Event) error

	tx.Wrapper
}

type Redis interface {
	Get(ctx context.Context, key string, target any) (domain.IdempotencyStatus, error)
	Set(ctx context.Context, key string, status domain.IdempotencyStatus, value any, ttl time.Duration) error
	TryLock(ctx context.Context, key string, ttl time.Duration) (bool, error)
	Delete(ctx context.Context, key string) error
}

type Usecase struct {
	postgres Postgres
	redis    Redis

	log zerolog.Logger
}

var usecase *Usecase

func New(
	postgres Postgres,
	redis Redis,
	log zerolog.Logger,
) *Usecase {
	usecase = &Usecase{
		postgres: postgres,
		redis:    redis,

		log: log.With().Str("component", "usecase: task.CreateTask").Logger(),
	}

	return usecase
}

func (u *Usecase) CreateTask(ctx context.Context, in Input) (Output, error) {
	log := u.log.With().Ctx(ctx).Logger()

	if err := in.Validate(); err != nil {
		log.Info().Err(err).Msg(domain.ErrInputValidation.Error())
		return Output{}, domain.ErrInputValidation
	}

	ok, err := u.redis.TryLock(ctx, in.IdempotencyKey, 3*time.Minute)
	if err != nil {
		log.Warn().Err(err).Msg("redis: try lock error")
		return Output{}, err
	}
	if !ok {
		var result Output
		status, err := u.redis.Get(ctx, in.IdempotencyKey, &result)
		if err != nil {
			log.Error().Err(err).Msg("redis: get key error")
			return Output{}, err
		}
		if status == domain.IdempotencyStatusCompleted {
			return result, nil
		}
		return Output{}, domain.ErrOperationInProgress
	}

	var success bool
	defer func() {
		if success {
			return
		}

		if err := u.redis.Delete(ctx, in.IdempotencyKey); err != nil {
			log.Error().Err(err).Msg("redis: cleanup key error")
		}
	}()

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

	out, err := u.createAndPublishTask(ctx, task, equation)

	if err != nil {
		return Output{}, err
	}
	success = true

	if err := u.redis.Set(ctx, in.IdempotencyKey, domain.IdempotencyStatusCompleted, out, time.Hour); err != nil {
		log.Warn().Err(err).Msg("redis: set ")
	}

	return out, nil
}

func (u *Usecase) createAndPublishTask(
	ctx context.Context,
	task domain.Task,
	equation domain.Equation,
) (Output, error) {
	log := u.log.With().Ctx(ctx).Logger()

	out := Output{
		Task:     task,
		Equation: equation,
	}

	err := u.postgres.Wrap(ctx, func(ctx context.Context) error {
		var err error

		out.Task, err = u.postgres.CreateTask(ctx, out.Task)
		if err != nil {
			log.Error().Err(err).Msg("usecase: postgres.CreateTask")
			return err
		}

		event, err := u.createTaskTrainEvent(out.Task)
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
		return Output{}, err
	}

	return out, nil
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
