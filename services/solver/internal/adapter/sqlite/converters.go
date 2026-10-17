package sqlite

import (
	"encoding/json"
	"fmt"

	"github.com/AridanWarlock/pinnAutomizer/pkg/core"
	shared "github.com/AridanWarlock/pinnAutomizer/pkg/sqlite"
	"github.com/AridanWarlock/pinnAutomizer/solver/internal/domain"
)

func newAttemptRow(a domain.Attempt) attemptRow {
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

func (r attemptRow) toDomain() domain.Attempt {
	a, err := domain.RestoreAttempt(domain.AttemptSnapshot{
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
	if err != nil {
		panic(fmt.Errorf("sqlite: invalid persisted attempt: %w", err))
	}
	return a
}

func newMessageRow(message core.KafkaMessage) messageRow {
	headers, err := json.Marshal(message.Headers)
	if err != nil {
		// KafkaHeaders is map[string]string; an encoding failure is an internal bug.
		panic(fmt.Errorf("sqlite: encode message headers: %w", err))
	}
	return messageRow{
		Topic:   message.Topic,
		Key:     message.Key,
		Value:   message.Value,
		Headers: string(headers),
	}
}

func (r messageRow) toDomain() core.KafkaMessage {
	var headers core.KafkaHeaders
	if err := json.Unmarshal([]byte(r.Headers), &headers); err != nil {
		panic(fmt.Errorf("sqlite: invalid persisted headers: %w", err))
	}
	message, err := core.NewProduceKafkaMessage(r.Topic, r.Key, r.Value, headers)
	if err != nil {
		panic(fmt.Errorf("sqlite: invalid persisted message: %w", err))
	}
	return message
}

func newEventRow(event domain.Event) eventRow {
	return eventRow{
		ID:         event.ID,
		RunID:      event.RunID,
		messageRow: newMessageRow(event.Message),
	}
}

func (r eventRow) toDomain() domain.Event {
	event, err := domain.NewEvent(r.ID, r.RunID, r.messageRow.toDomain())
	if err != nil {
		panic(fmt.Errorf("sqlite: invalid persisted event: %w", err))
	}
	return event
}
