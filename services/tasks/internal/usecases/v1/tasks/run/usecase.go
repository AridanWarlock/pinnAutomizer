package tasksRun

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/google/uuid"

	"github.com/AridanWarlock/pinnAutomizer/pkg/core"
	"github.com/AridanWarlock/pinnAutomizer/pkg/errs"
	"github.com/AridanWarlock/pinnAutomizer/pkg/postgres/poolx"
	"github.com/AridanWarlock/pinnAutomizer/tasks/internal/domain"
)

type Postgres interface {
	CreateTaskRunRequest(ctx context.Context, request domain.TaskRunRequest) (bool, error)
	GetTaskRunRequest(ctx context.Context, userID uuid.UUID, idKey core.IdempotencyKey) (domain.TaskRunRequest, error)
	GetTaskByIDAndUserIDForUpdate(ctx context.Context, id, userID uuid.UUID) (domain.Task, error)
	UpdateTaskStatusByID(ctx context.Context, id uuid.UUID, newStatus, oldStatus domain.TaskStatus) error
	PublishEvent(ctx context.Context, event domain.Event) (domain.Event, error)

	poolx.TxManager
}

type usecase struct {
	postgres Postgres
}

func New(postgres Postgres) Usecase {
	return &usecase{postgres: postgres}
}

func (u *usecase) RunTask(ctx context.Context, in Input) error {
	if err := in.Validate(); err != nil {
		return fmt.Errorf("%w: %v", errs.ErrInvalidArgument, err)
	}
	auth := core.MustAuthInfoFromContext(ctx)
	idKey := core.MustIdempotencyKeyFromContext(ctx)
	request, err := domain.NewTaskRunRequest(auth.UserID, in.TaskID, idKey)
	if err != nil {
		return fmt.Errorf("create run request: %w", err)
	}
	return u.postgres.InTransaction(ctx, func(ctx context.Context) error {
		created, err := u.postgres.CreateTaskRunRequest(ctx, request)
		if err != nil {
			return fmt.Errorf("save run request: %w", err)
		}
		if !created {
			existing, err := u.postgres.GetTaskRunRequest(ctx, auth.UserID, idKey)
			if err != nil {
				return fmt.Errorf("get accepted run request: %w", err)
			}
			if existing.TaskID != in.TaskID {
				return fmt.Errorf("%w: idempotency key was used for a different task", errs.ErrConflict)
			}
			return nil
		}
		return u.createAndPublishEvent(ctx, in.TaskID, auth.UserID, core.IdempotencyKey(request.EventID.String()))
	})
}

func (u *usecase) createAndPublishEvent(
	ctx context.Context,
	taskID, userID uuid.UUID,
	idKey core.IdempotencyKey,
) error {
	task, err := u.postgres.GetTaskByIDAndUserIDForUpdate(ctx, taskID, userID)
	if err != nil {
		return fmt.Errorf("getting task by id and user id: %w", err)
	}
	if task.IsStarted() {
		return domain.ErrTaskAlreadyStarted
	}

	err = u.postgres.UpdateTaskStatusByID(ctx, taskID, domain.TaskStatusInQueue, domain.TaskStatusCreated)
	if err != nil {
		return fmt.Errorf("update task status in postgres: %w", err)
	}

	event, err := u.createRunTaskEvent(task, idKey)
	if err != nil {
		return fmt.Errorf("create solve task event: %w", err)
	}

	_, err = u.postgres.PublishEvent(ctx, event)
	if err != nil {
		return fmt.Errorf("publish event in postgres: %w", err)
	}

	return nil
}

func (u *usecase) createRunTaskEvent(task domain.Task, idKey core.IdempotencyKey) (domain.Event, error) {
	msg, err := domain.NewRunTaskMessage(task)
	if err != nil {
		return domain.Event{}, fmt.Errorf("create run task message: %w", err)
	}

	jsonMsg, err := json.Marshal(msg)
	if err != nil {
		return domain.Event{}, fmt.Errorf("marshal run task message: %w", err)
	}

	var topic string
	switch task.Mode {
	case domain.TaskModeTrain, domain.TaskModeRetrain:
		topic = "tasks.run.queue"
	case domain.TaskModePredict:
		topic = "tasks.predict.queue"
	default:
		return domain.Event{}, fmt.Errorf("unexpected task mode: %s", task.Mode)
	}

	event, err := domain.NewEvent(idKey, topic, jsonMsg)
	if err != nil {
		return domain.Event{}, fmt.Errorf("create event: %w", err)
	}

	return event, nil
}
