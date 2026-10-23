package tasksCreate

import (
	"context"
	"errors"
	"os"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"

	"github.com/AridanWarlock/pinnAutomizer/pkg/core/corefixtures"
	"github.com/AridanWarlock/pinnAutomizer/tasks/internal/domain"
)

func TestCreationKeepsCleanupUntilFilesAndTransactionSucceed(t *testing.T) {
	t.Parallel()
	for _, failureAt := range []string{"", "reserve", "data", "output", "store", "insert", "confirm", "commit", "cancel"} {
		t.Run("failure="+failureAt, func(t *testing.T) {
			t.Parallel()
			failure := errors.New("injected failure")
			if failureAt == "cancel" {
				failure = context.Canceled
			}
			ctx, cancel := context.WithCancel(corefixtures.NewAuthInfo().WithContext(t.Context()))
			defer cancel()
			var inTransaction, locked, pending, taskCommitted bool
			var order []string
			operation := func(stage string) error {
				order = append(order, stage)
				if stage == failureAt {
					return failure
				}
				return nil
			}
			postgres := &MockPostgres{
				QueueFileCleanupFunc: func(_ context.Context, job domain.FileCleanup) error {
					require.True(t, locked)
					require.False(t, inTransaction)
					if err := operation("reserve"); err != nil {
						return err
					}
					pending = true
					return nil
				},
				InTransactionFunc: func(ctx context.Context, process func(context.Context) error) error {
					inTransaction = true
					defer func() { inTransaction = false }()
					err := process(ctx)
					if err != nil {
						return err
					}
					if err := operation("commit"); err != nil {
						return err
					}
					pending, taskCommitted = false, true
					return nil
				},
				CreateTaskFunc: func(_ context.Context, task domain.Task) (domain.Task, error) {
					require.True(t, locked)
					require.True(t, inTransaction)
					return task, operation("insert")
				},
				DeleteFileCleanupFunc: func(context.Context, uuid.UUID) error {
					require.True(t, inTransaction)
					return operation("confirm")
				},
			}
			mkdirCalls := 0
			files := &MockTaskFileStore{
				WithTaskFilesFunc: func(ctx context.Context, _ string, process func(context.Context) error) error {
					locked = true
					defer func() { locked = false }()
					return process(ctx)
				},
				MkdirAllFunc: func(string, os.FileMode) error {
					require.False(t, inTransaction, "filesystem I/O must not hold a SQL transaction")
					require.True(t, pending, "durable compensation must precede filesystem I/O")
					mkdirCalls++
					if mkdirCalls == 1 {
						return operation("data")
					}
					return operation("output")
				},
				StoreFunc: func(string, []domain.TaskFile) error {
					require.False(t, inTransaction)
					if failureAt == "cancel" {
						order = append(order, "cancel")
						cancel()
						return nil
					}
					return operation("store")
				},
			}
			out, err := New(postgres, files).CreateTask(ctx, Input{Name: "test", Mode: domain.TaskModeTrain})
			if failureAt == "" {
				require.NoError(t, err)
				require.NoError(t, out.Task.Validate())
				require.True(t, taskCommitted)
				require.False(t, pending)
			} else {
				require.ErrorIs(t, err, failure)
				require.False(t, taskCommitted)
				require.Equal(t, failureAt != "reserve", pending)
				require.Equal(t, failureAt, order[len(order)-1], "no effects after a failed stage")
			}
			require.False(t, locked)
			require.False(t, inTransaction)
		})
	}
}
