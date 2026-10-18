package tasksRun

import (
	"context"
	"encoding/json"
	"errors"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"

	"github.com/AridanWarlock/pinnAutomizer/pkg/core"
	"github.com/AridanWarlock/pinnAutomizer/pkg/core/corefixtures"
	"github.com/AridanWarlock/pinnAutomizer/pkg/errs"
	"github.com/AridanWarlock/pinnAutomizer/tasks/internal/domain"
)

type runFixture struct {
	ctx      context.Context
	task     domain.Task
	idKey    core.IdempotencyKey
	postgres *MockPostgres
}

func newRunFixture(t *testing.T) runFixture {
	t.Helper()
	auth := corefixtures.NewAuthInfo()
	idKey := corefixtures.NewIdempotencyKey()
	task, err := domain.NewTask("example", auth.UserID, domain.TaskModeTrain, nil)
	require.NoError(t, err)
	return runFixture{
		ctx:   idKey.WithContext(auth.WithContext(context.Background())),
		task:  task,
		idKey: idKey,
		postgres: &MockPostgres{
			InTransactionFunc: func(ctx context.Context, fn func(context.Context) error) error {
				return fn(ctx)
			},
		},
	}
}

func TestRunTaskInvalidInputDoesNotStartTransaction(t *testing.T) {
	t.Parallel()
	postgres := &MockPostgres{}
	err := New(postgres).RunTask(context.Background(), Input{})
	require.ErrorIs(t, err, errs.ErrInvalidArgument)
	require.Empty(t, postgres.InTransactionCalls())
}

func TestRunTaskReplayDoesNotReadTaskOrPublishAgain(t *testing.T) {
	t.Parallel()
	f := newRunFixture(t)
	f.postgres.CreateTaskRunRequestFunc = func(context.Context, domain.TaskRunRequest) (bool, error) {
		return false, nil
	}
	f.postgres.GetTaskRunRequestFunc = func(_ context.Context, userID uuid.UUID, idKey core.IdempotencyKey) (domain.TaskRunRequest, error) {
		require.Equal(t, f.task.UserID, userID)
		require.Equal(t, f.idKey, idKey)
		return domain.TaskRunRequest{TaskID: f.task.ID}, nil
	}
	err := New(f.postgres).RunTask(f.ctx, Input{TaskID: f.task.ID})
	require.NoError(t, err)
	require.Empty(t, f.postgres.GetTaskByIDAndUserIDForUpdateCalls())
	require.Empty(t, f.postgres.UpdateTaskStatusByIDCalls())
	require.Empty(t, f.postgres.PublishEventCalls())
}

func TestRunTaskRejectsKeyPreviouslyUsedForAnotherTask(t *testing.T) {
	t.Parallel()
	f := newRunFixture(t)
	f.postgres.CreateTaskRunRequestFunc = func(context.Context, domain.TaskRunRequest) (bool, error) {
		return false, nil
	}
	f.postgres.GetTaskRunRequestFunc = func(context.Context, uuid.UUID, core.IdempotencyKey) (domain.TaskRunRequest, error) {
		return domain.TaskRunRequest{TaskID: uuid.New()}, nil
	}
	err := New(f.postgres).RunTask(f.ctx, Input{TaskID: f.task.ID})
	require.ErrorIs(t, err, errs.ErrConflict)
	require.Empty(t, f.postgres.GetTaskByIDAndUserIDForUpdateCalls())
	require.Empty(t, f.postgres.PublishEventCalls())
}

func TestRunTaskReturnsRequestStorageFailures(t *testing.T) {
	for _, stage := range []string{"save request", "read replay"} {
		t.Run(stage, func(t *testing.T) {
			t.Parallel()
			f := newRunFixture(t)
			failure := errors.New("storage unavailable")
			f.postgres.CreateTaskRunRequestFunc = func(context.Context, domain.TaskRunRequest) (bool, error) {
				if stage == "save request" {
					return false, failure
				}
				return false, nil
			}
			f.postgres.GetTaskRunRequestFunc = func(context.Context, uuid.UUID, core.IdempotencyKey) (domain.TaskRunRequest, error) {
				return domain.TaskRunRequest{}, failure
			}
			err := New(f.postgres).RunTask(f.ctx, Input{TaskID: f.task.ID})
			require.ErrorIs(t, err, failure)
			require.Empty(t, f.postgres.GetTaskByIDAndUserIDForUpdateCalls())
			require.Empty(t, f.postgres.PublishEventCalls())
		})
	}
}

func TestRunTaskPublishesCommandForRequestedMode(t *testing.T) {
	for _, tc := range []struct {
		mode  domain.TaskMode
		topic string
	}{
		{domain.TaskModeTrain, "tasks.run.queue"},
		{domain.TaskModeRetrain, "tasks.run.queue"},
		{domain.TaskModePredict, "tasks.predict.queue"},
	} {
		t.Run(string(tc.mode), func(t *testing.T) {
			t.Parallel()
			f := newRunFixture(t)
			f.task.Mode = tc.mode
			inTransaction := false
			var accepted domain.TaskRunRequest
			f.postgres.InTransactionFunc = func(ctx context.Context, fn func(context.Context) error) error {
				inTransaction = true
				defer func() {
					inTransaction = false
				}()
				return fn(ctx)
			}
			f.postgres.CreateTaskRunRequestFunc = func(_ context.Context, request domain.TaskRunRequest) (bool, error) {
				require.True(t, inTransaction)
				require.Equal(t, f.task.UserID, request.UserID)
				require.Equal(t, f.task.ID, request.TaskID)
				require.Equal(t, f.idKey, request.IdKey)
				accepted = request
				return true, nil
			}
			f.postgres.GetTaskByIDAndUserIDForUpdateFunc = func(_ context.Context, id, userID uuid.UUID) (domain.Task, error) {
				require.True(t, inTransaction)
				require.Equal(t, f.task.ID, id)
				require.Equal(t, f.task.UserID, userID)
				return f.task, nil
			}
			f.postgres.UpdateTaskStatusByIDFunc = func(_ context.Context, id uuid.UUID, next, previous domain.TaskStatus) error {
				require.True(t, inTransaction)
				require.Equal(t, f.task.ID, id)
				require.Equal(t, domain.TaskStatusCreated, previous)
				require.Equal(t, domain.TaskStatusInQueue, next)
				return nil
			}
			f.postgres.PublishEventFunc = func(_ context.Context, event domain.Event) (domain.Event, error) {
				require.True(t, inTransaction)
				require.NotEqual(t, uuid.Nil, accepted.EventID)
				require.Equal(t, accepted.EventID.String(), event.IdKey.String())
				require.Equal(t, tc.topic, event.Topic)
				var message domain.RunTaskMessage
				require.NoError(t, json.Unmarshal(event.Data, &message))
				require.Equal(t, domain.RunTaskMessage{TaskID: f.task.ID, Mode: tc.mode}, message)
				return event, nil
			}
			require.NoError(t, New(f.postgres).RunTask(f.ctx, Input{TaskID: f.task.ID}))
			require.Len(t, f.postgres.UpdateTaskStatusByIDCalls(), 1)
			require.Len(t, f.postgres.PublishEventCalls(), 1)
		})
	}
}

func TestRunTaskDoesNotPublishWhenTaskCannotBeEnqueued(t *testing.T) {
	for _, status := range []domain.TaskStatus{domain.TaskStatusInQueue, domain.TaskStatusRunning, domain.TaskStatusDone, domain.TaskStatusError} {
		t.Run(string(status), func(t *testing.T) {
			t.Parallel()
			f := newRunFixture(t)
			f.task.Status = status
			f.postgres.CreateTaskRunRequestFunc = func(context.Context, domain.TaskRunRequest) (bool, error) {
				return true, nil
			}
			f.postgres.GetTaskByIDAndUserIDForUpdateFunc = func(context.Context, uuid.UUID, uuid.UUID) (domain.Task, error) {
				return f.task, nil
			}
			err := New(f.postgres).RunTask(f.ctx, Input{TaskID: f.task.ID})
			require.ErrorIs(t, err, domain.ErrTaskAlreadyStarted)
			require.Empty(t, f.postgres.UpdateTaskStatusByIDCalls())
			require.Empty(t, f.postgres.PublishEventCalls())
		})
	}
}

func TestRunTaskReturnsEnqueueAndCommitFailures(t *testing.T) {
	for _, stage := range []string{"read task", "change status", "save event", "commit"} {
		t.Run(stage, func(t *testing.T) {
			t.Parallel()
			f := newRunFixture(t)
			failure := errors.New("transaction failed")
			f.postgres.InTransactionFunc = func(ctx context.Context, fn func(context.Context) error) error {
				if err := fn(ctx); err != nil {
					return err
				}
				if stage == "commit" {
					return failure
				}
				return nil
			}
			f.postgres.CreateTaskRunRequestFunc = func(context.Context, domain.TaskRunRequest) (bool, error) {
				return true, nil
			}
			f.postgres.GetTaskByIDAndUserIDForUpdateFunc = func(context.Context, uuid.UUID, uuid.UUID) (domain.Task, error) {
				if stage == "read task" {
					return domain.Task{}, failure
				}
				return f.task, nil
			}
			f.postgres.UpdateTaskStatusByIDFunc = func(context.Context, uuid.UUID, domain.TaskStatus, domain.TaskStatus) error {
				if stage == "change status" {
					return failure
				}
				return nil
			}
			f.postgres.PublishEventFunc = func(_ context.Context, event domain.Event) (domain.Event, error) {
				if stage == "save event" {
					return domain.Event{}, failure
				}
				return event, nil
			}
			err := New(f.postgres).RunTask(f.ctx, Input{TaskID: f.task.ID})
			require.ErrorIs(t, err, failure)
			if stage == "read task" || stage == "change status" {
				require.Empty(t, f.postgres.PublishEventCalls())
			}
		})
	}
}
