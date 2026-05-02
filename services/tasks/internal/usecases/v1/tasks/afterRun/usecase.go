package tasksAfterRun

import (
	"context"
	"fmt"

	"github.com/AridanWarlock/pinnAutomizer/pkg/core"
	"github.com/AridanWarlock/pinnAutomizer/pkg/errs"
	"github.com/AridanWarlock/pinnAutomizer/pkg/logger"
	"github.com/AridanWarlock/pinnAutomizer/tasks/internal/domain"
	"github.com/google/uuid"
)

type Postgres interface {
	UpdateTaskStatusByID(ctx context.Context, id uuid.UUID, status domain.TaskStatus) error
	UpdateTaskStatusAndErrorByID(ctx context.Context, id uuid.UUID, status domain.TaskStatus, errorMsg string) error
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

func (u *usecase) UpdateTaskAfterTrain(ctx context.Context, in Input) error {
	log := logger.FromContext(ctx)

	if err := in.Validate(); err != nil {
		return fmt.Errorf("%w: %v", errs.ErrInvalidArgument, err)
	}

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

	if in.Error == nil {
		err = u.postgres.UpdateTaskStatusByID(ctx, in.ID, domain.TaskStatusDone)
	} else {
		err = u.postgres.UpdateTaskStatusAndErrorByID(ctx, in.ID, domain.TaskStatusError, *in.Error)
	}

	if err != nil {
		return fmt.Errorf("update task status: %w", err)
	}

	success = true

	err = u.redis.Set(ctx, idKey, core.IdempotencyStatusCompleted, nil)
	if err != nil {
		log.Warn().Err(err).Msg("redis: set key error")
	}

	return nil
}
