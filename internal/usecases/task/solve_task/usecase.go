package solve_task

import (
	"context"
	"encoding/json"
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
	GetTaskByIDAndUserID(ctx context.Context, id, userID uuid.UUID) (domain.Task, error)
	PublishEvent(ctx context.Context, event domain.Event) error
}

type Usecase struct {
	postgres Postgres
	redis    Redis

	log zerolog.Logger
}

var usecase *Usecase

func New(postgres Postgres, redis Redis, log zerolog.Logger) *Usecase {
	usecase = &Usecase{
		postgres: postgres,
		redis:    redis,

		log: log.With().Str("component", "usecase: task.SolveTask").Logger(),
	}

	return usecase
}

func (u *Usecase) SolveTask(ctx context.Context, in Input) error {
	log := u.log.With().Ctx(ctx).Logger()

	if err := in.Validate(); err != nil {
		log.Error().Err(err).Msg("usecase: input.Validate")
		return err
	}

	ok, err := u.redis.TryLock(ctx, in.IdempotencyKey, 3*time.Minute)
	if err != nil {
		log.Warn().Err(err).Msg("redis: try lock error")
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

	task, err := u.postgres.GetTaskByIDAndUserID(ctx, in.TaskID, in.UserID)
	if err != nil {
		log.Error().Err(err).Msg("usecase: postgres.GetTaskByIDAndUserID")
		return err
	}
	if task.Status != domain.TaskStatusDone {
		log.Error().Err(domain.ErrTaskNotTrained).Msg("usecase: domain.ErrTaskNotTrained")
		return domain.ErrTaskNotTrained
	}

	event, err := u.createSolveTaskEvent(task, in.Constants)
	if err != nil {
		log.Error().Err(err).Msg("usecase: createSolveTaskEvent")
		return err
	}

	err = u.postgres.PublishEvent(ctx, event)
	if err != nil {
		log.Error().Err(err).Msg("usecase: postgres.PublishEvent")
		return err
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
		return domain.Event{}, err
	}

	return domain.NewEvent("to-solve", jsonMsg), nil
}
