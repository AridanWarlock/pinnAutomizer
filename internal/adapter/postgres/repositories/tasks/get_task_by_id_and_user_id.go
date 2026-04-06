package tasks

import (
	"context"
	"fmt"

	"github.com/AridanWarlock/pinnAutomizer/internal/adapter/postgres/pgerr"
	. "github.com/AridanWarlock/pinnAutomizer/internal/adapter/postgres/schema"
	"github.com/AridanWarlock/pinnAutomizer/internal/domain"
	"github.com/AridanWarlock/pinnAutomizer/internal/errs"

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
		if pgerr.IsNotFound(err) {
			return domain.Task{}, fmt.Errorf(
				"task with id=%v and user id=%v: %w",
				id,
				userID,
				errs.ErrNotFound,
			)
		}
		return domain.Task{}, pgerr.ScanErr(err)
	}
	return ToModel(outRow), nil
}
