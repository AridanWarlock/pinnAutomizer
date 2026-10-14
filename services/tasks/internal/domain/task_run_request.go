package domain

import (
	"fmt"
	"time"

	"github.com/google/uuid"

	"github.com/AridanWarlock/pinnAutomizer/pkg/core"
	"github.com/AridanWarlock/pinnAutomizer/pkg/errs"
	"github.com/AridanWarlock/pinnAutomizer/pkg/validate"
)

const TaskRunRequestIdKeyMaxBytes = 256

// TaskRunRequest records an accepted run independently of outbox delivery and task deletion.
type TaskRunRequest struct {
	UserID    uuid.UUID           `validate:"required,uuid"`
	IdKey     core.IdempotencyKey `validate:"required"`
	TaskID    uuid.UUID           `validate:"required,uuid"`
	EventID   uuid.UUID           `validate:"required,uuid"`
	CreatedAt time.Time           `validate:"required"`
}

func NewTaskRunRequest(userID, taskID uuid.UUID, idKey core.IdempotencyKey) (TaskRunRequest, error) {
	request := TaskRunRequest{
		UserID:    userID,
		TaskID:    taskID,
		IdKey:     idKey,
		EventID:   uuid.New(),
		CreatedAt: time.Now(),
	}
	if err := request.Validate(); err != nil {
		return TaskRunRequest{}, fmt.Errorf("%w: %v", errs.ErrInvalidArgument, err)
	}
	return request, nil
}

func (r TaskRunRequest) Validate() error {
	if len(r.IdKey) > TaskRunRequestIdKeyMaxBytes {
		return fmt.Errorf("idempotency key exceeds %d bytes", TaskRunRequestIdKeyMaxBytes)
	}
	return validate.V.Struct(r)
}
