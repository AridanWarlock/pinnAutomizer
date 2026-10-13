package postgres

import (
	"context"

	"github.com/AridanWarlock/pinnAutomizer/pkg/core"
	"github.com/AridanWarlock/pinnAutomizer/tasks/internal/domain"
	sq "github.com/Masterminds/squirrel"
	"github.com/google/uuid"
)

// CreateTaskRunRequest waits for a competing insertion of the same user's key.
// It must share a transaction with enqueueing the task and saving its outbox event.
func (r *Repository) CreateTaskRunRequest(ctx context.Context, request domain.TaskRunRequest) (bool, error) {
	row := FromTaskRunRequestModel(request)
	query := r.sb.Insert(TaskRunRequestsTable).
		Columns(TaskRunRequestsColumns...).
		Values(row.Values()...).
		Suffix("ON CONFLICT (user_id, id_key) DO NOTHING")
	tag, err := r.pool.Execx(ctx, query)
	if err != nil {
		return false, err
	}
	return tag.RowsAffected() == 1, nil
}

func (r *Repository) GetTaskRunRequest(ctx context.Context, userID uuid.UUID, idKey core.IdempotencyKey) (domain.TaskRunRequest, error) {
	query := r.sb.Select(TaskRunRequestsColumns...).
		From(TaskRunRequestsTable).
		Where(sq.Eq{TaskRunRequestsUserID: userID, TaskRunRequestsIdKey: idKey})
	var row TaskRunRequestRow
	if err := r.pool.Getx(ctx, &row, query); err != nil {
		return domain.TaskRunRequest{}, err
	}
	return ToTaskRunRequestModel(row), nil
}
