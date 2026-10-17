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

// Created distinguishes a new durable command from a successful replay.
type Created bool

// Changed is true only for the caller that applied a durable transition.
type Changed bool

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
	a := Attempt{
		RunID:     runID,
		TaskID:    taskID,
		Mode:      mode,
		State:     AttemptAccepted,
		CreatedAt: now.UTC(),
	}
	if err := a.Validate(); err != nil {
		return Attempt{}, err
	}
	return a, nil
}

// AttemptSnapshot is untrusted persistence input, not an initialized entity.
type AttemptSnapshot Attempt

func RestoreAttempt(snapshot AttemptSnapshot) (Attempt, error) {
	a := Attempt(snapshot)
	if err := a.Validate(); err != nil {
		return Attempt{}, err
	}
	return a, nil
}

func (a Attempt) Validate() error {
	if err := validate.V.Struct(a); err != nil {
		return fmt.Errorf("%w: %v", errs.ErrInvalidArgument, err)
	}
	invalid := a.CreatedAt.IsZero()
	switch a.State {
	case AttemptAccepted:
		invalid = invalid || a.ContainerID != "" || a.Deadline != nil
	case AttemptContainerCreated:
		invalid = invalid || a.ContainerID == "" || a.Deadline != nil
	case AttemptStartRequested, AttemptRunning:
		invalid = invalid || a.ContainerID == "" || a.Deadline == nil || a.Deadline.IsZero()
	case AttemptFinished:
		invalid = invalid || a.CompletedAt == nil || a.CompletedAt.IsZero() ||
			(a.Error == nil && (a.ExitCode == nil || *a.ExitCode != 0))
	default:
		invalid = true
	}
	if a.State != AttemptFinished {
		invalid = invalid || a.CompletedAt != nil || a.ExitCode != nil || a.Error != nil
	}
	if invalid {
		return fmt.Errorf("%w: inconsistent attempt state", errs.ErrInvalidArgument)
	}
	return nil
}

func (a Attempt) StartAlreadyHandled() bool {
	return a.State == AttemptStartRequested || a.State == AttemptRunning || a.State == AttemptFinished
}

func (a *Attempt) RecordContainer(containerID string) (Changed, error) {
	if containerID == "" {
		return false, errs.ErrInvalidArgument
	}
	if a.State != AttemptAccepted {
		if a.ContainerID == containerID {
			return false, nil
		}
		return false, errs.ErrConflict
	}
	next := *a
	next.State = AttemptContainerCreated
	next.ContainerID = containerID
	return a.apply(next)
}

func (a *Attempt) RequestStart(now time.Time, timeout time.Duration) (Changed, error) {
	if now.IsZero() || timeout <= 0 {
		return false, errs.ErrInvalidArgument
	}
	if a.StartAlreadyHandled() {
		return false, nil
	}
	if a.State != AttemptContainerCreated {
		return false, errs.ErrConflict
	}
	next := *a
	deadline := now.Add(timeout).UTC()
	next.State = AttemptStartRequested
	next.Deadline = &deadline
	return a.apply(next)
}

func (a *Attempt) Finish(code *int, failure *string, now time.Time) (Changed, error) {
	if now.IsZero() || (failure == nil && (code == nil || *code != 0)) {
		return false, errs.ErrInvalidArgument
	}
	if a.State == AttemptFinished {
		return false, nil
	}
	next := *a
	completed := now.UTC()
	next.State = AttemptFinished
	next.CompletedAt = &completed
	if code != nil {
		value := *code
		next.ExitCode = &value
	}
	if failure != nil {
		value := *failure
		next.Error = &value
	}
	return a.apply(next)
}

func (a *Attempt) apply(next Attempt) (Changed, error) {
	if err := next.Validate(); err != nil {
		return false, err
	}
	*a = next
	return true, nil
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

func NewEvent(id, runID uuid.UUID, message core.KafkaMessage) (Event, error) {
	e := Event{ID: id, RunID: runID, Message: message}
	if err := e.Validate(); err != nil {
		return Event{}, err
	}
	return e, nil
}

func (e Event) Validate() error {
	if err := validate.V.Struct(e); err != nil {
		return fmt.Errorf("%w: %v", errs.ErrInvalidArgument, err)
	}
	if err := e.Message.ValidateOutgoing(); err != nil {
		return err
	}
	idKey, err := e.Message.IdKey()
	if err != nil || idKey.String() != e.ID.String() {
		return fmt.Errorf("%w: event identity differs from message identity", errs.ErrInvalidArgument)
	}
	return nil
}
