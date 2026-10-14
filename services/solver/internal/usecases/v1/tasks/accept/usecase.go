package tasksAccept

import (
	"context"
	"fmt"
	"time"

	"github.com/AridanWarlock/pinnAutomizer/solver/internal/domain"
)

type usecase struct{ repo Repository }

func New(repo Repository) Usecase { return &usecase{repo: repo} }

// AcceptTask persists the command before the transport may acknowledge it.
func (u *usecase) AcceptTask(ctx context.Context, in Input) (Output, error) {
	a, err := domain.NewAttempt(in.RunID, in.TaskID, in.Mode, time.Now())
	if err != nil {
		return Output{}, err
	}
	stored, created, err := u.repo.RegisterAttempt(ctx, a)
	if err != nil {
		return Output{}, fmt.Errorf("register solver attempt: %w", err)
	}
	return Output{Attempt: stored, Created: created}, nil
}
