package domain

import (
	"fmt"
	"time"

	"github.com/google/uuid"

	"github.com/AridanWarlock/pinnAutomizer/pkg/core"
	"github.com/AridanWarlock/pinnAutomizer/pkg/errs"
)

type AttemptState string

const (
	AttemptRegistered     AttemptState = "registered"
	AttemptCreated        AttemptState = "created"
	AttemptStartRequested AttemptState = "start_requested"
	AttemptRunning        AttemptState = "running"
	AttemptFinished       AttemptState = "finished"
)

type Attempt struct {
	RunID       uuid.UUID
	TaskID      uuid.UUID
	Mode        string
	State       AttemptState
	ContainerID string
	CreatedAt   time.Time
	Deadline    *time.Time
	CompletedAt *time.Time
	ExitCode    *int
	Error       *string
}

func NewAttempt(runID, taskID uuid.UUID, mode string, now time.Time) (Attempt, error) {
	if runID == uuid.Nil || now.IsZero() {
		return Attempt{}, fmt.Errorf("%w: invalid run ID or timestamp", errs.ErrInvalidArgument)
	}
	if _, err := NewMlTask(taskID, mode); err != nil {
		return Attempt{}, fmt.Errorf("%w: %v", errs.ErrInvalidArgument, err)
	}
	return Attempt{RunID: runID, TaskID: taskID, Mode: mode, State: AttemptRegistered, CreatedAt: now.UTC()}, nil
}

func (a Attempt) ContainerName() string { return "pinn-solver-" + a.RunID.String() }
func (a Attempt) Matches(other Attempt) bool {
	return a.RunID == other.RunID && a.TaskID == other.TaskID && a.Mode == other.Mode
}

// Event preserves a stable identity and wire payload across outbox retries.
type Event struct {
	ID      uuid.UUID
	RunID   uuid.UUID
	Message core.KafkaMessage
}

func (e Event) Validate() error {
	if e.ID == uuid.Nil || e.RunID == uuid.Nil || e.Message.Topic == "" || len(e.Message.Value) == 0 ||
		e.Message.Headers["X-Idempotency-Key"] != e.ID.String() ||
		e.Message.Partition != 0 || e.Message.Offset != 0 || !e.Message.Time.IsZero() {
		return fmt.Errorf("%w: invalid solver event", errs.ErrInvalidArgument)
	}
	return nil
}
