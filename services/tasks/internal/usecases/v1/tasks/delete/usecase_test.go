package tasksDelete

import (
	"context"
	"errors"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/AridanWarlock/pinnAutomizer/pkg/core/corefixtures"
	"github.com/AridanWarlock/pinnAutomizer/pkg/test"
	"github.com/AridanWarlock/pinnAutomizer/tasks/internal/domain"
)

func TestDeleteTaskDoesNotRemoveFilesOnCommitFailure(t *testing.T) {
	failure := errors.New("commit failed")
	auth := corefixtures.NewAuthInfo()
	task, err := domain.NewTask("task", auth.UserID, domain.TaskModeTrain, nil)
	require.NoError(t, err)
	postgres := &MockPostgres{
		InTransactionFunc: func(ctx context.Context, fn func(context.Context) error) error {
			if err := fn(ctx); err != nil {
				return err
			}
			return failure
		},
		GetTaskByIDAndUserIDForUpdateFunc: func(ctx context.Context, id, userID uuid.UUID) (domain.Task, error) {
			return task, nil
		},
		DeleteTaskByIDAndUserIDFunc: func(ctx context.Context, id, userID uuid.UUID) error {
			return nil
		},
	}
	files := &MockFileService{}
	ctx := auth.WithContext(test.ContextWithZeroLogger())
	err = New(postgres, files).DeleteTask(ctx, Input{TaskID: task.ID})
	require.ErrorIs(t, err, failure)
	assert.Empty(t, files.RemoveAllCalls())
}

func TestDeleteTaskRemovesFilesAfterCommit(t *testing.T) {
	auth := corefixtures.NewAuthInfo()
	task, err := domain.NewTask("task", auth.UserID, domain.TaskModeTrain, nil)
	require.NoError(t, err)
	committed := false
	postgres := &MockPostgres{
		InTransactionFunc: func(ctx context.Context, fn func(context.Context) error) error {
			if err := fn(ctx); err != nil {
				return err
			}
			committed = true
			return nil
		},
		GetTaskByIDAndUserIDForUpdateFunc: func(ctx context.Context, id, userID uuid.UUID) (domain.Task, error) {
			return task, nil
		},
		DeleteTaskByIDAndUserIDFunc: func(ctx context.Context, id, userID uuid.UUID) error {
			return nil
		},
	}
	files := &MockFileService{RemoveAllFunc: func(dir string) error {
		assert.True(t, committed)
		return nil
	}}
	ctx := auth.WithContext(test.ContextWithZeroLogger())
	require.NoError(t, New(postgres, files).DeleteTask(ctx, Input{TaskID: task.ID}))
	require.Len(t, files.RemoveAllCalls(), 2)
	assert.Equal(t, task.DataPath, files.RemoveAllCalls()[0].Dir)
	assert.Equal(t, task.OutputPath, files.RemoveAllCalls()[1].Dir)
}
