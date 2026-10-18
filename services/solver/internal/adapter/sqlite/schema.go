package sqlite

import (
	"github.com/google/uuid"

	shared "github.com/AridanWarlock/pinnAutomizer/pkg/sqlite"
	"github.com/AridanWarlock/pinnAutomizer/solver/internal/domain"
)

const attemptsTable = "attempts"

const (
	attemptsRunID       = "run_id"
	attemptsTaskID      = "task_id"
	attemptsMode        = "mode"
	attemptsState       = "state"
	attemptsContainerID = "container_id"
	attemptsCreatedAt   = "created_at"
	attemptsDeadline    = "deadline"
	attemptsCompletedAt = "completed_at"
	attemptsExitCode    = "exit_code"
	attemptsError       = "error"
)

var attemptColumns = []string{
	attemptsRunID, attemptsTaskID, attemptsMode, attemptsState, attemptsContainerID,
	attemptsCreatedAt, attemptsDeadline, attemptsCompletedAt, attemptsExitCode, attemptsError,
}

type attemptRow struct {
	RunID       uuid.UUID           `db:"run_id"`
	TaskID      uuid.UUID           `db:"task_id"`
	Mode        string              `db:"mode"`
	State       domain.AttemptState `db:"state"`
	ContainerID string              `db:"container_id"`
	CreatedAt   shared.SQLTime      `db:"created_at"`
	Deadline    *shared.SQLTime     `db:"deadline"`
	CompletedAt *shared.SQLTime     `db:"completed_at"`
	ExitCode    *int                `db:"exit_code"`
	Error       *string             `db:"error"`
}

func (r attemptRow) Values() []any {
	return []any{
		r.RunID, r.TaskID, r.Mode, r.State, r.ContainerID,
		r.CreatedAt, r.Deadline, r.CompletedAt, r.ExitCode, r.Error,
	}
}

const outboxTable = "outbox"

const (
	outboxID       = "event_id"
	outboxRunID    = "run_id"
	outboxTopic    = "topic"
	outboxKey      = "message_key"
	outboxValue    = "value"
	outboxHeaders  = "headers"
	outboxSequence = "sequence"
)

var eventColumns = []string{
	outboxID, outboxRunID, outboxTopic, outboxKey, outboxValue, outboxHeaders,
}

type messageRow struct {
	Topic   string `db:"topic"`
	Key     []byte `db:"message_key"`
	Value   []byte `db:"value"`
	Headers string `db:"headers"`
}

func (r messageRow) Values() []any {
	return []any{r.Topic, r.Key, r.Value, r.Headers}
}

type eventRow struct {
	ID    uuid.UUID `db:"event_id"`
	RunID uuid.UUID `db:"run_id"`
	messageRow
}

func (r eventRow) Values() []any {
	return append([]any{r.ID, r.RunID}, r.messageRow.Values()...)
}
