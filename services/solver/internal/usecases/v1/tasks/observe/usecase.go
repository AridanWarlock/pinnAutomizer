package tasksObserve

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"

	"github.com/AridanWarlock/pinnAutomizer/pkg/core"
	"github.com/AridanWarlock/pinnAutomizer/solver/internal/domain"
)

type SQLite interface {
	GetAttempt(ctx context.Context, id uuid.UUID) (domain.Attempt, error)
	FinishAttempt(ctx context.Context, id uuid.UUID, code *int, failure *string, now time.Time, event domain.Event) (core.Changed, error)
}

type PinnRunner interface {
	ObserveAttempt(ctx context.Context, attempt domain.Attempt) (domain.Attempt, error)
}

type usecase struct {
	sqlite     SQLite
	pinnRunner PinnRunner
}

func New(sqlite SQLite, pinnRunner PinnRunner) Usecase {
	return &usecase{
		sqlite:     sqlite,
		pinnRunner: pinnRunner,
	}
}

func (u *usecase) ObserveTask(ctx context.Context, in Input) (Output, error) {
	if err := in.Validate(); err != nil {
		return Output{}, err
	}
	a, err := u.sqlite.GetAttempt(ctx, in.RunID)
	if err != nil {
		return Output{}, fmt.Errorf("get attempt: %w", err)
	}
	if !a.CanObserve() {
		return Output{}, nil
	}
	observed, err := u.pinnRunner.ObserveAttempt(ctx, a)
	if err != nil {
		return Output{}, fmt.Errorf("observe attempt: %w", err)
	}
	if !observed.IsFinished() {
		return Output{}, nil
	}
	event, err := newCompletionEvent(observed)
	if err != nil {
		return Output{}, err
	}
	changed, err := u.sqlite.FinishAttempt(ctx, observed.RunID, observed.ExitCode, observed.Error, *observed.CompletedAt, event)
	if err != nil {
		return Output{}, fmt.Errorf("finish attempt: %w", err)
	}
	return Output{Finished: changed}, nil
}
