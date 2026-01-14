package tasks

import (
	"context"
	. "pinnAutomizer/internal/adapter/postgres/schema"
	"pinnAutomizer/internal/domain"

	. "github.com/Masterminds/squirrel"
	"github.com/google/uuid"
)

func (r *Repository) GetTaskByIDAndUserID(
	ctx context.Context,
	id uuid.UUID,
	userID uuid.UUID,
) (domain.Task, error) {
	query := r.sb.
		Select(TasksTableColumns...).
		From(TasksTable).
		Where(Eq{TasksTableColumnID: id, TasksTableColumnUserId: userID})

	var outRow TaskRow
	if err := r.pool.Getx(ctx, &outRow, query); err != nil {
		return domain.Task{}, err
	}
	return ToModel(outRow), nil
}
