package tasksRecover

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"

	"github.com/AridanWarlock/pinnAutomizer/pkg/core/limit"
	"github.com/AridanWarlock/pinnAutomizer/pkg/errs"
	"github.com/AridanWarlock/pinnAutomizer/solver/internal/domain"
	tasksObserve "github.com/AridanWarlock/pinnAutomizer/solver/internal/usecases/v1/tasks/observe"
	tasksStart "github.com/AridanWarlock/pinnAutomizer/solver/internal/usecases/v1/tasks/start"
)

type SQLite interface {
	GetUnfinishedAttempts(ctx context.Context, afterRunID uuid.UUID, batch limit.Limit) ([]domain.Attempt, error)
}

type TasksStart interface {
	StartTask(ctx context.Context, in tasksStart.Input) (tasksStart.Output, error)
}

type TasksObserve interface {
	ObserveTask(ctx context.Context, in tasksObserve.Input) (tasksObserve.Output, error)
}

type usecase struct {
	sqlite         SQLite
	tasksStart     TasksStart
	tasksObserve   TasksObserve
	batch          limit.Limit
	attemptTimeout time.Duration
}

func New(sqlite SQLite, start TasksStart, observe TasksObserve, batch limit.Limit, attemptTimeout time.Duration) (Usecase, error) {
	if attemptTimeout <= 0 {
		return nil, fmt.Errorf("%w: positive attempt timeout required", errs.ErrInvalidArgument)
	}
	return &usecase{
		sqlite:         sqlite,
		tasksStart:     start,
		tasksObserve:   observe,
		batch:          batch,
		attemptTimeout: attemptTimeout,
	}, nil
}

func (u *usecase) RecoverPending(ctx context.Context, in Input) (Output, error) {
	out := Output{NextRunID: in.AfterRunID}
	if err := ctx.Err(); err != nil {
		return out, err
	}
	attempts, err := u.sqlite.GetUnfinishedAttempts(ctx, in.AfterRunID, u.batch)
	if err != nil {
		return out, fmt.Errorf("list unfinished attempts: %w", err)
	}
	if len(attempts) == 0 {
		return Output{}, nil
	}
	var failures error
	for _, a := range attempts {
		if err := ctx.Err(); err != nil {
			return out, errors.Join(failures, err)
		}
		if err := u.recoverAttempt(ctx, a); err != nil {
			failures = errors.Join(failures, fmt.Errorf("recover run %s: %w", a.RunID, err))
		}
		out.NextRunID = a.RunID
	}
	return out, failures
}

func (u *usecase) recoverAttempt(ctx context.Context, a domain.Attempt) error {
	ctx, cancel := context.WithTimeout(ctx, u.attemptTimeout)
	defer cancel()
	if !a.StartAlreadyHandled() {
		if _, err := u.tasksStart.StartTask(ctx, tasksStart.Input{RunID: a.RunID}); err != nil {
			return err
		}
	}
	_, err := u.tasksObserve.ObserveTask(ctx, tasksObserve.Input{RunID: a.RunID})
	return err
}
