package postgres

import (
	"time"

	"github.com/AridanWarlock/pinnAutomizer/pkg/core"
	"github.com/AridanWarlock/pinnAutomizer/tasks/internal/domain"
	"github.com/google/uuid"
)

type TaskRunRequestRow struct {
	UserID    uuid.UUID           `db:"user_id"`
	IdKey     core.IdempotencyKey `db:"id_key"`
	TaskID    uuid.UUID           `db:"task_id"`
	EventID   uuid.UUID           `db:"event_id"`
	CreatedAt time.Time           `db:"created_at"`
}

func (r TaskRunRequestRow) Values() []any {
	return []any{r.UserID, r.IdKey, r.TaskID, r.EventID, r.CreatedAt}
}

func FromTaskRunRequestModel(request domain.TaskRunRequest) TaskRunRequestRow {
	return TaskRunRequestRow{
		UserID:    request.UserID,
		IdKey:     request.IdKey,
		TaskID:    request.TaskID,
		EventID:   request.EventID,
		CreatedAt: request.CreatedAt,
	}
}

func ToTaskRunRequestModel(row TaskRunRequestRow) domain.TaskRunRequest {
	return domain.TaskRunRequest{
		UserID:    row.UserID,
		IdKey:     row.IdKey,
		TaskID:    row.TaskID,
		EventID:   row.EventID,
		CreatedAt: row.CreatedAt,
	}
}
