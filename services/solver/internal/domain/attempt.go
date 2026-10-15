package domain

import (
	"fmt"
	"time"

	"github.com/google/uuid"

	"github.com/AridanWarlock/pinnAutomizer/pkg/core"
	"github.com/AridanWarlock/pinnAutomizer/pkg/errs"
	"github.com/AridanWarlock/pinnAutomizer/pkg/validate"
)

type AttemptState string

// Stored values remain compatible with the existing SQL migration.
const (
	// AttemptAccepted means the command is durably recorded; no container exists yet.
	AttemptAccepted AttemptState = "registered"
	// AttemptContainerCreated means a container ID is persisted; it has not been started.
	AttemptContainerCreated AttemptState = "created"
	AttemptStartRequested   AttemptState = "start_requested"
	AttemptRunning          AttemptState = "running"
	AttemptFinished         AttemptState = "finished"
)

type Attempt struct {
	RunID       uuid.UUID `validate:"required,uuid"`
	TaskID      uuid.UUID `validate:"required,uuid"`
	Mode        string    `validate:"required,oneof=train retrain predict"`
	State       AttemptState
	ContainerID string
	CreatedAt   time.Time `validate:"required"`
	Deadline    *time.Time
	CompletedAt *time.Time
	ExitCode    *int
	Error       *string
}

func NewAttempt(runID, taskID uuid.UUID, mode string, now time.Time) (Attempt, error) {
	a := Attempt{RunID: runID, TaskID: taskID, Mode: mode, State: AttemptAccepted, CreatedAt: now.UTC()}
	if err := validate.V.Struct(a); err != nil {
		return Attempt{}, fmt.Errorf("%w: %v", errs.ErrInvalidArgument, err)
	}
	return a, nil
}

func (a Attempt) Matches(other Attempt) bool {
	return a.RunID == other.RunID && a.TaskID == other.TaskID && a.Mode == other.Mode
}

// Event preserves a stable identity and wire payload across outbox retries.
type Event struct {
	ID      uuid.UUID `validate:"required,uuid"`
	RunID   uuid.UUID `validate:"required,uuid"`
	Message core.KafkaMessage
}

func (e Event) Validate() error {
	if err := validate.V.Struct(e); err != nil {
		return fmt.Errorf("%w: %v", errs.ErrInvalidArgument, err)
	}
	if err := validate.V.Var(e.Message.Topic, "required"); err != nil {
		return fmt.Errorf("%w: event topic: %v", errs.ErrInvalidArgument, err)
	}
	if err := validate.V.Var(e.Message.Value, "required,min=1"); err != nil {
		return fmt.Errorf("%w: event payload: %v", errs.ErrInvalidArgument, err)
	}
	if e.Message.Headers["X-Idempotency-Key"] != e.ID.String() || e.Message.Partition != 0 || e.Message.Offset != 0 || !e.Message.Time.IsZero() {
		return fmt.Errorf("%w: invalid outgoing event metadata", errs.ErrInvalidArgument)
	}
	return nil
}
