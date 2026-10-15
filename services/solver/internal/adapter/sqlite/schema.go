package sqlite

import (
	"time"

	"github.com/google/uuid"

	"github.com/AridanWarlock/pinnAutomizer/solver/internal/domain"
)

const attemptsTable = "attempts"
const outboxTable = "outbox"

var attemptColumns = []string{"run_id", "task_id", "mode", "state", "container_id", "created_at", "deadline", "completed_at", "exit_code", "error"}
var eventColumns = []string{"event_id", "run_id", "topic", "message_key", "value", "headers"}

type attemptRow struct {
	RunID       uuid.UUID           `db:"run_id"`
	TaskID      uuid.UUID           `db:"task_id"`
	Mode        string              `db:"mode"`
	State       domain.AttemptState `db:"state"`
	ContainerID string              `db:"container_id"`
	CreatedAt   int64               `db:"created_at"`
	Deadline    *int64              `db:"deadline"`
	CompletedAt *int64              `db:"completed_at"`
	ExitCode    *int                `db:"exit_code"`
	Error       *string             `db:"error"`
}

func (r attemptRow) toDomain() domain.Attempt {
	return domain.Attempt{RunID: r.RunID, TaskID: r.TaskID, Mode: r.Mode, State: r.State, ContainerID: r.ContainerID, CreatedAt: time.Unix(0, r.CreatedAt).UTC(), Deadline: timestamp(r.Deadline), CompletedAt: timestamp(r.CompletedAt), ExitCode: r.ExitCode, Error: r.Error}
}

func timestamp(nanoseconds *int64) *time.Time {
	if nanoseconds == nil {
		return nil
	}
	value := time.Unix(0, *nanoseconds).UTC()
	return &value
}

type eventRow struct {
	ID      uuid.UUID `db:"event_id"`
	RunID   uuid.UUID `db:"run_id"`
	Topic   string    `db:"topic"`
	Key     []byte    `db:"message_key"`
	Value   []byte    `db:"value"`
	Headers string    `db:"headers"`
}

func fromAttempt(a domain.Attempt) attemptRow {
	return attemptRow{RunID: a.RunID, TaskID: a.TaskID, Mode: a.Mode, State: a.State, ContainerID: a.ContainerID, CreatedAt: a.CreatedAt.UnixNano(), Deadline: nanoseconds(a.Deadline), CompletedAt: nanoseconds(a.CompletedAt), ExitCode: a.ExitCode, Error: a.Error}
}

func (r attemptRow) Values() []any {
	return []any{r.RunID.String(), r.TaskID.String(), r.Mode, r.State, r.ContainerID, r.CreatedAt, r.Deadline, r.CompletedAt, r.ExitCode, r.Error}
}

func nanoseconds(timestamp *time.Time) *int64 {
	if timestamp == nil {
		return nil
	}
	value := timestamp.UnixNano()
	return &value
}
