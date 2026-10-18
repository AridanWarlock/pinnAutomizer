package tasksStart

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
	RecordContainer(ctx context.Context, id uuid.UUID, containerID string) (core.Changed, error)
	RequestStart(ctx context.Context, id uuid.UUID, now time.Time, timeout time.Duration) (core.Changed, error)
}

type PinnRunner interface {
	EnsureContainer(ctx context.Context, attempt domain.Attempt) (string, error)
	StartContainer(ctx context.Context, attempt domain.Attempt) error
}

type usecase struct {
	sqlite     SQLite
	pinnRunner PinnRunner
	timeout    time.Duration
}

func New(sqlite SQLite, pinnRunner PinnRunner, timeout time.Duration) Usecase {
	return &usecase{
		sqlite:     sqlite,
		pinnRunner: pinnRunner,
		timeout:    timeout,
	}
}

func (u *usecase) StartTask(ctx context.Context, in Input) (Output, error) {
	if err := in.Validate(); err != nil {
		return Output{}, err
	}
	a, err := u.sqlite.GetAttempt(ctx, in.RunID)
	if err != nil {
		return Output{}, fmt.Errorf("get attempt: %w", err)
	}
	if a.StartAlreadyHandled() {
		return Output{}, nil
	}
	if a.State == domain.AttemptAccepted {
		id, err := u.pinnRunner.EnsureContainer(ctx, a)
		if err != nil {
			return Output{}, fmt.Errorf("ensure container: %w", err)
		}
		if _, err := u.sqlite.RecordContainer(ctx, a.RunID, id); err != nil {
			return Output{}, fmt.Errorf("record container: %w", err)
		}
	}
	changed, err := u.sqlite.RequestStart(ctx, a.RunID, time.Now(), u.timeout)
	if err != nil {
		return Output{}, fmt.Errorf("request start: %w", err)
	}
	out := Output{StartRequested: changed}
	if !changed {
		return out, nil
	}
	// Reload the committed snapshot. No transaction spans Docker I/O.
	a, err = u.sqlite.GetAttempt(ctx, a.RunID)
	if err != nil {
		return out, fmt.Errorf("reload start intent: %w", err)
	}
	if err := u.pinnRunner.StartContainer(ctx, a); err != nil {
		return out, fmt.Errorf("start attempt: %w", err)
	}
	return out, nil
}
