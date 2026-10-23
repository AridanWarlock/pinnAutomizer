package filecleanup

import (
	"context"
	"errors"
	"testing"
	"testing/synctest"

	"github.com/google/uuid"
	"github.com/rs/zerolog"
	"github.com/stretchr/testify/require"

	tasksCleanup "github.com/AridanWarlock/pinnAutomizer/tasks/internal/usecases/v1/tasks/cleanup"
)

func TestWorkerAdvancesPastFailureResetsSweepAndStops(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		ctx, cancel := context.WithCancel(t.Context())
		defer cancel()
		cursor := uuid.New()
		var inputs []tasksCleanup.Input
		usecase := &MockUsecase{
			CleanupPendingFunc: func(_ context.Context, in tasksCleanup.Input) (tasksCleanup.Output, error) {
				inputs = append(inputs, in)
				switch len(inputs) {
				case 1:
					return tasksCleanup.Output{NextTaskID: cursor}, errors.New("one job failed")
				case 3:
					cancel()
				}
				return tasksCleanup.Output{}, nil
			},
		}
		done := make(chan error, 1)
		go func() { done <- NewWorker(usecase, zerolog.Nop()).Run(ctx) }()
		require.ErrorIs(t, <-done, context.Canceled)
		require.Equal(t, []tasksCleanup.Input{{}, {AfterTaskID: cursor}, {}}, inputs)
	})
}
