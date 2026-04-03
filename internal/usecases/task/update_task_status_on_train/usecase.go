package update_task_status_on_train

import (
	"context"
	"pinnAutomizer/internal/domain"
	"time"

	"github.com/google/uuid"
	"github.com/rs/zerolog"
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

		log: log.With().Str("component", "usecase: task.UpdateTaskStatusOnTrain").Logger(),
	}

	return usecase
}

func (u *Usecase) UpdateTaskStatusOnTrain(ctx context.Context, in Input) error {
	log := u.log.With().Ctx(ctx).Logger()

	if err := in.Validate(); err != nil {
		log.Error().Err(err).Msg("input validation failed")
		return err
	}

	ok, err := u.redis.TryLock(ctx, in.IdempotencyKey, 3*time.Minute)
	if err != nil {
		log.Warn().Err(err).Msg("redis: try lock key error")
		return err
	}
	if !ok {
		status, err := u.redis.Get(ctx, in.IdempotencyKey, nil)
		if err != nil {
			log.Warn().Err(err).Msg("redis: get key error")
			return err
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

	err = u.postgres.UpdateTaskStatusByID(
		ctx,
		in.ID,
		string(domain.TaskStatusTraining),
		string(domain.TaskStatusCreated),
	)
	if err != nil {
		log.Error().Err(err).Msg("postgres updating task status error")
		return err
	}

	success = true

	err = u.redis.Set(ctx, in.IdempotencyKey, domain.IdempotencyStatusCompleted, nil, 24*time.Hour)
	if err != nil {
		log.Warn().Err(err).Msg("redis: set key error")
	}

	return nil
}
