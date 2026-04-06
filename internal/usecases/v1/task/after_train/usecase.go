package tasks_after_train

import (
	"context"
	"fmt"
	"time"

	"github.com/AridanWarlock/pinnAutomizer/internal/domain"
	"github.com/AridanWarlock/pinnAutomizer/internal/errs"
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
	UpdateTaskStatusByID(ctx context.Context, id uuid.UUID, status, oldStatus string) error
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

	err = u.postgres.UpdateTaskStatusByID(ctx, in.ID, string(domain.TaskStatusDone), string(domain.TaskStatusTraining))
	if err != nil {
		return fmt.Errorf("update task status: %w", err)
	}

	success = true

	err = u.redis.Set(ctx, in.IdempotencyKey, domain.IdempotencyStatusCompleted, nil, 24*time.Hour)
	if err != nil {
		log.Warn().Err(err).Msg("redis: set key error")
	}

	return nil
}
