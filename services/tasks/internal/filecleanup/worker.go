package filecleanup

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/rs/zerolog"

	"github.com/AridanWarlock/pinnAutomizer/pkg/worker"
	tasksCleanup "github.com/AridanWarlock/pinnAutomizer/tasks/internal/usecases/v1/tasks/cleanup"
)

const retryInterval = 5 * time.Second

type Usecase interface {
	CleanupPending(ctx context.Context, in tasksCleanup.Input) (tasksCleanup.Output, error)
}

type Worker struct {
	usecase Usecase
	log     zerolog.Logger

	cursor uuid.UUID
}

func NewWorker(usecase Usecase, log zerolog.Logger) *Worker {
	return &Worker{
		usecase: usecase,
		log:     log,
	}
}

func (w *Worker) process(ctx context.Context) error {
	in := tasksCleanup.Input{
		AfterTaskID: w.cursor,
	}
	out, err := w.usecase.CleanupPending(ctx, in)

	w.cursor = out.NextTaskID
	return err
}

func (w *Worker) Run(ctx context.Context) error {
	loop, err := worker.New(retryInterval, w.log, w.process)
	if err != nil {
		return err
	}

	return loop.Run(ctx)
}
