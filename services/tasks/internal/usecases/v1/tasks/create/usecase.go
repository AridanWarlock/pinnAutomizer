package tasksCreate

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/AridanWarlock/pinnAutomizer/pkg/core"
	"github.com/AridanWarlock/pinnAutomizer/pkg/errs"
	"github.com/AridanWarlock/pinnAutomizer/tasks/internal/domain"
)

type Postgres interface {
	CreateTask(ctx context.Context, task domain.Task) (domain.Task, error)
	PublishEvent(ctx context.Context, event domain.Event) (domain.Event, error)
	InTransaction(ctx context.Context, inTx func(ctx context.Context) error) error
}

type TaskFileStore interface {
	Store(task domain.Task, files []domain.TaskFile) error
}

type usecase struct {
	postgres  Postgres
	fileStore TaskFileStore
}

func New(
	postgres Postgres,
	fileStore TaskFileStore,
) Usecase {
	return &usecase{
		postgres:  postgres,
		fileStore: fileStore,
	}
}

func (u *usecase) CreateTask(ctx context.Context, in Input) (Output, error) {
	authInfo := core.MustAuthInfoFromContext(ctx)

	if err := in.Validate(); err != nil {
		return Output{}, fmt.Errorf("%w: %v", errs.ErrInvalidArgument, err)
	}

	task, err := domain.NewTask(
		in.Name,
		authInfo.UserID,
		in.Mode,
		in.Description,
	)
	if err != nil {
		return Output{}, fmt.Errorf("create task model: %w", err)
	}

	err = u.postgres.InTransaction(ctx, func(ctx context.Context) error {
		task, err = u.createAndPublishTask(ctx, task, in)
		return err
	})

	if err != nil {
		return Output{}, fmt.Errorf("create task transaction: %w", err)
	}

	return Output{Task: task}, nil
}

func (u *usecase) createAndPublishTask(
	ctx context.Context,
	task domain.Task,
	in Input,
) (domain.Task, error) {
	task, err := u.postgres.CreateTask(ctx, task)
	if err != nil {
		return domain.Task{}, fmt.Errorf("create task in postgres: %w", err)
	}

	err = u.fileStore.Store(task, in.Files)
	if err != nil {
		return domain.Task{}, fmt.Errorf("store files: %w", err)
	}

	event, err := u.createTaskTrainEvent(task)
	if err != nil {
		return domain.Task{}, fmt.Errorf("create task train event in postgres: %w", err)
	}

	_, err = u.postgres.PublishEvent(ctx, event)
	if err != nil {
		return domain.Task{}, fmt.Errorf("publish event in postgres: %w", err)
	}

	return task, nil
}

func (u *usecase) createTaskTrainEvent(task domain.Task) (domain.Event, error) {
	msg := domain.TrainMessage{
		TaskID: task.ID,
		Mode:   task.Mode,
	}

	jsonMsg, err := json.Marshal(msg)
	if err != nil {
		return domain.Event{}, fmt.Errorf("marshal train message: %w", err)
	}

	return domain.NewEvent("tasks.on.run", jsonMsg), nil
}
