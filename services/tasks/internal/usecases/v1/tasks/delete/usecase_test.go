package tasksDelete

import (
	"context"
	"errors"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"

	"github.com/AridanWarlock/pinnAutomizer/pkg/core/corefixtures"
	"github.com/AridanWarlock/pinnAutomizer/pkg/errs"
	"github.com/AridanWarlock/pinnAutomizer/pkg/test"
	"github.com/AridanWarlock/pinnAutomizer/tasks/internal/domain"
)

func TestDeleteTaskQueuesCleanupInsideDeleteTransaction(t *testing.T) {
	t.Parallel()
	for _, failureAt := range []string{"none", "queue", "delete", "commit"} {
		t.Run(failureAt, func(t *testing.T) {
			auth := corefixtures.NewAuthInfo()
			task, err := domain.NewTask("task", auth.UserID, domain.TaskModeTrain, nil)
			require.NoError(t, err)
			failure := errors.New("injected failure")
			inTx := false
			var steps []string
			postgres := &MockPostgres{
				InTransactionFunc: func(ctx context.Context, fn func(context.Context) error) error {
					inTx = true
					defer func() { inTx = false }()
					if err := fn(ctx); err != nil {
						return err
					}
					if failureAt == "commit" {
						return failure
					}
					return nil
				},
				GetTaskByIDAndUserIDForUpdateFunc: func(context.Context, uuid.UUID, uuid.UUID) (domain.Task, error) {
					return task, nil
				},
				QueueFileCleanupFunc: func(_ context.Context, job domain.FileCleanup) error {
					require.True(t, inTx)
					require.Equal(t, task.ID, job.TaskID)
					require.Equal(t, task.DataPath, job.DataPath)
					require.Equal(t, task.OutputPath, job.OutputPath)
					steps = append(steps, "queue")
					if failureAt == "queue" {
						return failure
					}
					return nil
				},
				DeleteTaskByIDAndUserIDFunc: func(context.Context, uuid.UUID, uuid.UUID) error {
					require.True(t, inTx)
					steps = append(steps, "delete")
					if failureAt == "delete" {
						return failure
					}
					return nil
				},
			}
			ctx := auth.WithContext(test.ContextWithZeroLogger())
			err = New(postgres).DeleteTask(ctx, Input{TaskID: task.ID})
			if failureAt == "none" {
				require.NoError(t, err)
			} else {
				require.ErrorIs(t, err, failure)
			}
			if failureAt == "queue" {
				require.Equal(t, []string{"queue"}, steps)
			} else {
				require.Equal(t, []string{"queue", "delete"}, steps)
			}
		})
	}
}

func TestDeleteMissingTaskIsAnIdempotentNoop(t *testing.T) {
	t.Parallel()
	postgres := &MockPostgres{
		InTransactionFunc: func(ctx context.Context, fn func(context.Context) error) error { return fn(ctx) },
		GetTaskByIDAndUserIDForUpdateFunc: func(context.Context, uuid.UUID, uuid.UUID) (domain.Task, error) {
			return domain.Task{}, errs.ErrNotFound
		},
	}
	auth := corefixtures.NewAuthInfo()
	ctx := auth.WithContext(test.ContextWithZeroLogger())
	require.NoError(t, New(postgres).DeleteTask(ctx, Input{TaskID: uuid.New()}))
	require.Empty(t, postgres.QueueFileCleanupCalls())
	require.Empty(t, postgres.DeleteTaskByIDAndUserIDCalls())
}
