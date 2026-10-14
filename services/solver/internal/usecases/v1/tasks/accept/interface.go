package tasksAccept

import (
	"context"

	"github.com/google/uuid"

	"github.com/AridanWarlock/pinnAutomizer/solver/internal/domain"
)

type Input struct {
	RunID  uuid.UUID
	TaskID uuid.UUID
	Mode   string
}
type Output struct {
	Attempt domain.Attempt
	Created bool
}
type SQLite interface {
	RegisterAttempt(context.Context, domain.Attempt) (domain.Attempt, bool, error)
}
type Usecase interface {
	AcceptTask(context.Context, Input) (Output, error)
}
