package tasksAccept

import (
	"context"
	"fmt"
	"time"

	"github.com/AridanWarlock/pinnAutomizer/pkg/core"
	"github.com/AridanWarlock/pinnAutomizer/solver/internal/domain"
)

type SQLite interface {
	RegisterAttempt(ctx context.Context, attempt domain.Attempt) (stored domain.Attempt, created core.Created, err error)
}

type usecase struct {
	sqlite SQLite
}

func New(sqlite SQLite) Usecase {
	return &usecase{sqlite: sqlite}
}

// AcceptTask persists the command before the transport may acknowledge it.
func (u *usecase) AcceptTask(ctx context.Context, in Input) (Output, error) {
	if err := in.Validate(); err != nil {
		return Output{}, err
	}
	a, err := domain.NewAttempt(in.RunID, in.TaskID, in.Mode, time.Now())
	if err != nil {
		return Output{}, err
	}
	stored, created, err := u.sqlite.RegisterAttempt(ctx, a)
	if err != nil {
		return Output{}, fmt.Errorf("register solver attempt: %w", err)
	}
	return Output{Attempt: stored, Created: created}, nil
}
