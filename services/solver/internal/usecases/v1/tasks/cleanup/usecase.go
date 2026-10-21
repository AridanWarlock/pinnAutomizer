package tasksCleanup

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"

	"github.com/AridanWarlock/pinnAutomizer/pkg/core"
	"github.com/AridanWarlock/pinnAutomizer/pkg/errs"
	"github.com/AridanWarlock/pinnAutomizer/solver/internal/domain"
)

type SQLite interface {
	GetAttempt(ctx context.Context, id uuid.UUID) (domain.Attempt, error)
	MarkCleaned(ctx context.Context, id uuid.UUID, now time.Time) (core.Changed, error)
}

type PinnRunner interface {
	CleanupAttempt(ctx context.Context, attempt domain.Attempt) error
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

func (u *usecase) CleanupTask(ctx context.Context, in Input) (Output, error) {
	if err := in.Validate(); err != nil {
		return Output{}, err
	}
	a, err := u.sqlite.GetAttempt(ctx, in.RunID)
	if err != nil {
		return Output{}, fmt.Errorf("get cleanup attempt: %w", err)
	}
	if !a.IsFinished() {
		return Output{}, errs.ErrConflict
	}
	if !a.NeedsCleanup() {
		return Output{}, nil
	}
	if err := u.pinnRunner.CleanupAttempt(ctx, a); err != nil {
		return Output{}, fmt.Errorf("remove attempt container: %w", err)
	}
	changed, err := u.sqlite.MarkCleaned(ctx, a.RunID, time.Now())
	if err != nil {
		return Output{}, fmt.Errorf("record cleanup: %w", err)
	}
	return Output{Cleaned: changed}, nil
}
