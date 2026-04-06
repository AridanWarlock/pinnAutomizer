package tasks

import (
	"context"

	"github.com/AridanWarlock/pinnAutomizer/internal/adapter/postgres/pgerr"
	. "github.com/AridanWarlock/pinnAutomizer/internal/adapter/postgres/schema"
	"github.com/AridanWarlock/pinnAutomizer/internal/domain"

	. "github.com/Masterminds/squirrel"
	"github.com/google/uuid"
)

func (r *Repository) GetTasksByIDs(
	ctx context.Context,
	ids []uuid.UUID,
	userID uuid.UUID,
) ([]domain.Task, error) {
	query := r.sb.
		Select(TasksTableColumns...).
		From(TasksTable).
		Where(Eq{TasksTableColumnUserId: userID}).
		Where(Eq{TasksTableColumnID: ids})

	var rows []TaskRow
	if err := r.pool.Selectx(ctx, &rows, query); err != nil {
		return nil, pgerr.ScanErr(err)
	}

	tasks := make([]domain.Task, len(rows))
	for i, row := range rows {
		tasks[i] = ToModel(row)
	}
	return tasks, nil
}
