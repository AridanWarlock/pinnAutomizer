package tasksCleanup

import (
	"context"
	"errors"
	"path/filepath"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"

	"github.com/AridanWarlock/pinnAutomizer/pkg/core/limit"
	"github.com/AridanWarlock/pinnAutomizer/tasks/internal/domain"
)

func job(t *testing.T) domain.FileCleanup {
	t.Helper()
	id := uuid.New()
	job, err := domain.NewFileCleanup(id, filepath.Join("/tasks_data", id.String()), filepath.Join("/tasks_output", id.String()))
	require.NoError(t, err)
	return job
}

func TestCleanupRetriesPartialFailureAndDoesNotStarveNextJob(t *testing.T) {
	t.Parallel()
	first, second := job(t), job(t)
	failure := errors.New("filesystem unavailable")
	var removed []string
	var deleted []uuid.UUID
	postgres := &MockPostgres{
		HasFileCleanupFunc: func(context.Context, uuid.UUID) (bool, error) { return true, nil },
		GetFileCleanupsFunc: func(context.Context, uuid.UUID, limit.Limit) ([]domain.FileCleanup, error) {
			return []domain.FileCleanup{first, second}, nil
		},
		DeleteFileCleanupFunc: func(_ context.Context, id uuid.UUID) error {
			deleted = append(deleted, id)
			return nil
		},
	}
	files := &MockFileService{
		WithTaskFilesFunc: func(ctx context.Context, _ string, process func(context.Context) error) error { return process(ctx) },
		RemoveAllFunc: func(path string) error {
			removed = append(removed, path)
			if path == first.DataPath {
				return failure
			}
			return nil
		},
	}
	uc := New(postgres, files, limit.MustNew(2))
	out, err := uc.CleanupPending(t.Context(), Input{})
	require.ErrorIs(t, err, failure)
	require.ErrorContains(t, err, first.TaskID.String())
	require.Equal(t, second.TaskID, out.NextTaskID)
	require.Equal(t, []uuid.UUID{second.TaskID}, deleted)
	require.Equal(t, []string{first.DataPath, first.OutputPath, second.DataPath, second.OutputPath}, removed)
	// Advance to an empty page: the worker will use its Nil output for a new sweep.
	postgres.GetFileCleanupsFunc = func(_ context.Context, after uuid.UUID, _ limit.Limit) ([]domain.FileCleanup, error) {
		require.Equal(t, second.TaskID, after)
		return nil, nil
	}
	out, err = uc.CleanupPending(t.Context(), Input{AfterTaskID: out.NextTaskID})
	require.NoError(t, err)
	require.Equal(t, uuid.Nil, out.NextTaskID)
	// The retained first job is selected again, rather than forgotten behind the cursor.
	files.RemoveAllFunc = func(path string) error { return nil }
	postgres.GetFileCleanupsFunc = func(_ context.Context, after uuid.UUID, _ limit.Limit) ([]domain.FileCleanup, error) {
		require.Equal(t, uuid.Nil, after)
		return []domain.FileCleanup{first}, nil
	}
	_, err = uc.CleanupPending(t.Context(), Input{AfterTaskID: out.NextTaskID})
	require.NoError(t, err)
	require.Equal(t, []uuid.UUID{second.TaskID, first.TaskID}, deleted)
}

func TestCleanupRetainsCursorOnListingFailureAndDoesNotAcknowledgeOnCancellation(t *testing.T) {
	t.Parallel()
	cursor := uuid.New()
	failure := errors.New("database unavailable")
	postgres := &MockPostgres{
		HasFileCleanupFunc:  func(context.Context, uuid.UUID) (bool, error) { return true, nil },
		GetFileCleanupsFunc: func(context.Context, uuid.UUID, limit.Limit) ([]domain.FileCleanup, error) { return nil, failure },
	}
	files := &MockFileService{
		WithTaskFilesFunc: func(ctx context.Context, _ string, process func(context.Context) error) error { return process(ctx) },
	}
	out, err := New(postgres, files, limit.MustNew(1)).CleanupPending(t.Context(), Input{AfterTaskID: cursor})
	require.ErrorIs(t, err, failure)
	require.Equal(t, cursor, out.NextTaskID)
	require.Empty(t, files.RemoveAllCalls())

	j := job(t)
	ctx, cancel := context.WithCancel(t.Context())
	defer cancel()
	postgres.GetFileCleanupsFunc = func(context.Context, uuid.UUID, limit.Limit) ([]domain.FileCleanup, error) {
		return []domain.FileCleanup{j}, nil
	}
	files.RemoveAllFunc = func(string) error { cancel(); return nil }
	_, err = New(postgres, files, limit.MustNew(1)).CleanupPending(ctx, Input{})
	require.ErrorIs(t, err, context.Canceled)
	require.Len(t, files.RemoveAllCalls(), 1)
	require.Empty(t, postgres.DeleteFileCleanupCalls())
}
