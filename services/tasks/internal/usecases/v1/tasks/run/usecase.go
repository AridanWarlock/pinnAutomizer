package tasksRun

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/AridanWarlock/pinnAutomizer/pkg/core"
	"github.com/AridanWarlock/pinnAutomizer/pkg/errs"
	"github.com/AridanWarlock/pinnAutomizer/pkg/logger"
	"github.com/AridanWarlock/pinnAutomizer/tasks/internal/domain"
	"github.com/google/uuid"
)

type Postgres interface {
	GetTaskByIDAndUserID(ctx context.Context, id, userID uuid.UUID) (domain.Task, error)
	UpdateTaskStatusByID(ctx context.Context, id uuid.UUID, status domain.TaskStatus) error
	PublishEvent(ctx context.Context, event domain.Event) (domain.Event, error)

	InTransaction(ctx context.Context, inTx func(ctx context.Context) error) error
}

type Redis interface {
	Get(ctx context.Context, idKey core.IdempotencyKey, target any) (core.IdempotencyStatus, error)
	Set(ctx context.Context, idKey core.IdempotencyKey, status core.IdempotencyStatus, value any) error
	TryLock(ctx context.Context, idKey core.IdempotencyKey) (bool, error)
	Delete(ctx context.Context, idKey core.IdempotencyKey) error
}

type usecase struct {
	postgres Postgres
	redis    Redis
}

func New(
	postgres Postgres,
	redis Redis,
) Usecase {
	return &usecase{
		postgres: postgres,
		redis:    redis,
	}
}

func (u *usecase) RunTask(ctx context.Context, in Input) error {
	log := logger.FromContext(ctx)

	if err := in.Validate(); err != nil {
		return fmt.Errorf("%w: %v", errs.ErrInvalidArgument, err)
	}
	auth := core.MustAuthInfoFromContext(ctx)

	idKey := core.MustIdempotencyKeyFromContext(ctx)
	ok, err := u.redis.TryLock(ctx, idKey)
	if err != nil {
		return fmt.Errorf("redis try lock: %w", err)
	}
	if !ok {
		status, err := u.redis.Get(ctx, idKey, nil)
		if err != nil {
			return fmt.Errorf("redis get key: %w", err)
		}
		if status == core.IdempotencyStatusCompleted {
			return nil
		}

		return errs.ErrOperationInProgress
	}

	var success bool
	defer func() {
		if success {
			return
		}

		if err := u.redis.Delete(ctx, idKey); err != nil {
			log.Error().Err(err).Msg("redis: cleanup key error")
		}
	}()

	err = u.postgres.InTransaction(ctx, func(ctx context.Context) error {
		return u.createAndPublishEvent(ctx, in.TaskID, auth.UserID)
	})
	if err != nil {
		return err
	}

	success = true

	if err := u.redis.Set(ctx, idKey, core.IdempotencyStatusCompleted, nil); err != nil {
		log.Warn().Err(err).Msg("redis: set key error")
	}

	return nil
}

func (u *usecase) createAndPublishEvent(ctx context.Context, taskID, userID uuid.UUID) error {
	task, err := u.postgres.GetTaskByIDAndUserID(ctx, taskID, userID)
	if err != nil {
		return fmt.Errorf("getting task by id and user id: %w", err)
	}
	if task.IsStarted() {
		return domain.ErrTaskAlreadyStarted
	}

	err = u.postgres.UpdateTaskStatusByID(ctx, taskID, domain.TaskStatusRunning)
	if err != nil {
		return fmt.Errorf("update task status in postgres: %w", err)
	}

	event, err := u.createRunTaskEvent(task)
	if err != nil {
		return fmt.Errorf("create solve task event: %w", err)
	}

	_, err = u.postgres.PublishEvent(ctx, event)
	if err != nil {
		return fmt.Errorf("publish event in postgres: %w", err)
	}

	return nil
}

func (u *usecase) createRunTaskEvent(task domain.Task) (domain.Event, error) {
	msg, err := domain.NewRunTaskMessage(task)
	if err != nil {
		return domain.Event{}, fmt.Errorf("create run task message: %w", err)
	}

	jsonMsg, err := json.Marshal(msg)
	if err != nil {
		return domain.Event{}, fmt.Errorf("marshal run task message: %w", err)
	}

	return domain.NewEvent("tasks.on.run", jsonMsg), nil
}
