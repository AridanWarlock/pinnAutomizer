package filecleanup

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/rs/zerolog"

	tasksCleanup "github.com/AridanWarlock/pinnAutomizer/tasks/internal/usecases/v1/tasks/cleanup"
)

const retryInterval = 5 * time.Second

type Usecase interface {
	CleanupPending(ctx context.Context, in tasksCleanup.Input) (tasksCleanup.Output, error)
}

type Worker struct {
	usecase Usecase
	log     zerolog.Logger
}

func NewWorker(usecase Usecase, log zerolog.Logger) *Worker {
	return &Worker{
		usecase: usecase,
		log:     log,
	}
}

func (w *Worker) Run(ctx context.Context) error {
	timer := time.NewTimer(0)
	defer timer.Stop()
	cursor := uuid.Nil
	for {
		if err := ctx.Err(); err != nil {
			return err
		}
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-timer.C:
		}
		out, err := w.usecase.CleanupPending(ctx, tasksCleanup.Input{AfterTaskID: cursor})
		if ctx.Err() != nil {
			return ctx.Err()
		}
		if err != nil {
			w.log.Error().Err(err).Msg("retry task file cleanup")
		}
		cursor = out.NextTaskID
		timer.Reset(retryInterval)
	}
}
