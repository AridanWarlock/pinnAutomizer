package sqlite

import (
	"encoding/json"
	"strings"

	"github.com/google/uuid"

	"github.com/AridanWarlock/pinnAutomizer/pkg/core"
	shared "github.com/AridanWarlock/pinnAutomizer/pkg/sqlite"
	"github.com/AridanWarlock/pinnAutomizer/solver/internal/domain"
)

const (
	attemptsTable       = "attempts"
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
	outboxTable         = "outbox"
	outboxID            = "event_id"
	outboxRunID         = "run_id"
	outboxTopic         = "topic"
	outboxKey           = "message_key"
	outboxValue         = "value"
	outboxHeaders       = "headers"
	outboxSequence      = "sequence"
)

var attemptColumns = []string{
	attemptsRunID, attemptsTaskID, attemptsMode, attemptsState, attemptsContainerID,
	attemptsCreatedAt, attemptsDeadline, attemptsCompletedAt, attemptsExitCode, attemptsError,
}

var eventColumns = []string{
	outboxID, outboxRunID, outboxTopic, outboxKey, outboxValue, outboxHeaders,
}

func joinColumns(columns []string) string {
	return strings.Join(columns, ",")
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

func (r attemptRow) toDomain() (domain.Attempt, error) {
	return domain.RestoreAttempt(domain.AttemptSnapshot{
		RunID:       r.RunID,
		TaskID:      r.TaskID,
		Mode:        r.Mode,
		State:       r.State,
		ContainerID: r.ContainerID,
		CreatedAt:   r.CreatedAt.Time,
		Deadline:    shared.Time(r.Deadline),
		CompletedAt: shared.Time(r.CompletedAt),
		ExitCode:    r.ExitCode,
		Error:       r.Error,
	})
}

func fromAttempt(a domain.Attempt) attemptRow {
	return attemptRow{
		RunID:       a.RunID,
		TaskID:      a.TaskID,
		Mode:        a.Mode,
		State:       a.State,
		ContainerID: a.ContainerID,
		CreatedAt:   shared.Timestamp(a.CreatedAt),
		Deadline:    shared.NullableTimestamp(a.Deadline),
		CompletedAt: shared.NullableTimestamp(a.CompletedAt),
		ExitCode:    a.ExitCode,
		Error:       a.Error,
	}
}

func (r attemptRow) Values() []any {
	return []any{
		r.RunID, r.TaskID, r.Mode, r.State, r.ContainerID,
		r.CreatedAt, r.Deadline, r.CompletedAt, r.ExitCode, r.Error,
	}
}

type messageRow struct {
	Topic   string `db:"topic"`
	Key     []byte `db:"message_key"`
	Value   []byte `db:"value"`
	Headers string `db:"headers"`
}

func (r messageRow) toDomain() (core.KafkaMessage, error) {
	var headers core.KafkaHeaders
	if err := json.Unmarshal([]byte(r.Headers), &headers); err != nil {
		return core.KafkaMessage{}, err
	}
	return core.NewProduceKafkaMessage(r.Topic, r.Key, r.Value, headers)
}

type eventRow struct {
	ID    uuid.UUID `db:"event_id"`
	RunID uuid.UUID `db:"run_id"`
	messageRow
}

func (r eventRow) toDomain() (domain.Event, error) {
	message, err := r.messageRow.toDomain()
	if err != nil {
		return domain.Event{}, err
	}
	return domain.NewEvent(r.ID, r.RunID, message)
}

func fromEvent(event domain.Event) (eventRow, error) {
	headers, err := json.Marshal(event.Message.Headers)
	if err != nil {
		return eventRow{}, err
	}
	return eventRow{
		ID:    event.ID,
		RunID: event.RunID,
		messageRow: messageRow{
			Topic:   event.Message.Topic,
			Key:     event.Message.Key,
			Value:   event.Message.Value,
			Headers: string(headers),
		},
	}, nil
}

func (r eventRow) Values() []any {
	return []any{r.ID, r.RunID, r.Topic, r.Key, r.Value, r.Headers}
}
