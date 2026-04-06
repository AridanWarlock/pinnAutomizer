package tasks_solve

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/AridanWarlock/pinnAutomizer/internal/domain"
	"github.com/AridanWarlock/pinnAutomizer/pkg/logger"
	"github.com/google/uuid"
)

type Redis interface {
	Get(ctx context.Context, key string, target any) (domain.IdempotencyStatus, error)
	Set(ctx context.Context, key string, status domain.IdempotencyStatus, value any, ttl time.Duration) error
	TryLock(ctx context.Context, key string, ttl time.Duration) (bool, error)
	Delete(ctx context.Context, key string) error
}

type Postgres interface {
	GetTaskByIDAndUserID(ctx context.Context, id, userID uuid.UUID) (domain.Task, error)
	PublishEvent(ctx context.Context, event domain.Event) error
}

type Usecase struct {
	postgres Postgres
	redis    Redis
}

func New(postgres Postgres, redis Redis) *Usecase {
	return &Usecase{
		postgres: postgres,
		redis:    redis,
	}
}

func (u *Usecase) SolveTask(ctx context.Context, in Input) error {
	log := logger.FromContext(ctx)

	if err := in.Validate(); err != nil {
		return domain.ErrInputValidation
	}

	ok, err := u.redis.TryLock(ctx, in.IdempotencyKey, 3*time.Minute)
	if err != nil {
		return fmt.Errorf("redis try lock: %w", err)
	}
	if !ok {
		status, err := u.redis.Get(ctx, in.IdempotencyKey, nil)
		if err != nil {
			return fmt.Errorf("redis get key: %w", err)
		}
		if status == domain.IdempotencyStatusCompleted {
			return nil
		}

		return domain.ErrOperationInProgress
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

	task, err := u.postgres.GetTaskByIDAndUserID(ctx, in.TaskID, in.UserID)
	if err != nil {
		return fmt.Errorf("getting task by id and user id: %w", err)
	}
	if task.Status != domain.TaskStatusDone {
		return domain.ErrTaskNotTrained
	}

	event, err := u.createSolveTaskEvent(task, in.Constants)
	if err != nil {
		return fmt.Errorf("create solve task event: %w", err)
	}

	err = u.postgres.PublishEvent(ctx, event)
	if err != nil {
		return fmt.Errorf("publish event in postgres: %w", err)
	}

	success = true

	if err := u.redis.Set(ctx, in.IdempotencyKey, domain.IdempotencyStatusCompleted, nil, 24*time.Hour); err != nil {
		log.Warn().Err(err).Msg("redis: set key error")
	}

	return nil
}

func (u *Usecase) createSolveTaskEvent(task domain.Task, solveConstants map[string]any) (domain.Event, error) {
	msg := domain.SolveTaskMessage{
		TaskID:    task.ID,
		ModelPath: task.ResultsPath,
		Constants: solveConstants,
	}

	jsonMsg, err := json.Marshal(msg)
	if err != nil {
		return domain.Event{}, fmt.Errorf("marshal solve task message: %w", err)
	}

	return domain.NewEvent("to-solve", jsonMsg), nil
}
