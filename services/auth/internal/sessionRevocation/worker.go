package sessionRevocation

import (
	"context"
	"fmt"
	"time"

	"github.com/rs/zerolog"

	"github.com/AridanWarlock/pinnAutomizer/auth/internal/usecases/v1/auth/cleanupSessions"
	"github.com/AridanWarlock/pinnAutomizer/pkg/core"
	"github.com/AridanWarlock/pinnAutomizer/pkg/errs"
	"github.com/AridanWarlock/pinnAutomizer/pkg/validate"
	"github.com/AridanWarlock/pinnAutomizer/pkg/worker"
)

type Config struct {
	Interval time.Duration `env:"INTERVAL" envDefault:"1s" validate:"gt=0"`
	Timeout  time.Duration `env:"TIMEOUT" envDefault:"5s" validate:"gt=0"`
}

type Usecase interface {
	Cleanup(ctx context.Context, in cleanupSessions.Input) (cleanupSessions.Output, error)
}

type BackgroundWorker interface {
	worker.Worker
}

type Worker struct {
	usecase Usecase
	worker  BackgroundWorker
	cursor  *core.Jti
	timeout time.Duration
}

func NewWorker(
	usecase Usecase,
	cfg Config,
	log zerolog.Logger,
) (*Worker, error) {
	if err := validate.V.Struct(cfg); err != nil {
		return nil, fmt.Errorf("%w: session revocation worker config: %w", errs.ErrInvalidArgument, err)
	}
	w := &Worker{
		usecase: usecase,
		timeout: cfg.Timeout,
	}
	loop, err := worker.New(cfg.Interval, log, w.process)
	if err != nil {
		return nil, err
	}
	w.worker = loop
	return w, nil
}

func (w *Worker) process(ctx context.Context) error {
	ctx, cancel := context.WithTimeout(ctx, w.timeout)
	defer cancel()
	out, err := w.usecase.Cleanup(ctx, cleanupSessions.Input{AfterJti: w.cursor})
	w.cursor = out.NextJti
	return err
}

func (w *Worker) Run(ctx context.Context) error {
	return w.worker.Run(ctx)
}
