package tasksSolve

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
	PublishEvent(ctx context.Context, event domain.Event) (domain.Event, error)
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

func (u *usecase) SolveTask(ctx context.Context, in Input) error {
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

	task, err := u.postgres.GetTaskByIDAndUserID(ctx, in.TaskID, auth.UserID)
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

	_, err = u.postgres.PublishEvent(ctx, event)
	if err != nil {
		return fmt.Errorf("publish event in postgres: %w", err)
	}

	success = true

	if err := u.redis.Set(ctx, idKey, core.IdempotencyStatusCompleted, nil); err != nil {
		log.Warn().Err(err).Msg("redis: set key error")
	}

	return nil
}

func (u *usecase) createSolveTaskEvent(task domain.Task, solveConstants map[string]any) (domain.Event, error) {
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
