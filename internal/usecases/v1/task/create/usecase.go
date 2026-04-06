package tasks_create

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/AridanWarlock/pinnAutomizer/internal/domain"
	"github.com/AridanWarlock/pinnAutomizer/pkg/logger"
	"github.com/AridanWarlock/pinnAutomizer/pkg/tx"
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
}

func New(
	postgres Postgres,
	redis Redis,
) *Usecase {
	return &Usecase{
		postgres: postgres,
		redis:    redis,
	}
}

func (u *Usecase) CreateTask(ctx context.Context, in Input) (Output, error) {
	log := logger.FromContext(ctx)

	if err := in.Validate(); err != nil {
		return Output{}, domain.ErrInputValidation
	}

	ok, err := u.redis.TryLock(ctx, in.IdempotencyKey, 3*time.Minute)
	if err != nil {
		return Output{}, fmt.Errorf("redis try lock: %w", err)
	}
	if !ok {
		var result Output
		status, err := u.redis.Get(ctx, in.IdempotencyKey, &result)
		if err != nil {
			return Output{}, fmt.Errorf("redis get key: %w", err)
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
		return Output{}, fmt.Errorf("getting equation from postgres: %w", err)
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
		return Output{}, fmt.Errorf("create task model: %w", err)
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
	out := Output{
		Task:     task,
		Equation: equation,
	}

	err := u.postgres.Wrap(ctx, func(ctx context.Context) error {
		var err error

		out.Task, err = u.postgres.CreateTask(ctx, out.Task)
		if err != nil {
			return fmt.Errorf("create task in postgres: %w", err)
		}

		event, err := u.createTaskTrainEvent(out.Task)
		if err != nil {
			return fmt.Errorf("create task train event in postgres: %w", err)
		}

		err = u.postgres.PublishEvent(ctx, event)
		if err != nil {
			return fmt.Errorf("publish event in postgres: %w", err)
		}

		return nil
	})

	if err != nil {
		return Output{}, fmt.Errorf("create task transaction: %w", err)
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
		return domain.Event{}, fmt.Errorf("marshal train message: %w", err)
	}

	return domain.NewEvent("to-train", jsonMsg), nil
}
