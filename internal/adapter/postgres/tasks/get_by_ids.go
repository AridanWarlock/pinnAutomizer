package tasks

import (
	"context"
	sq "github.com/Masterminds/squirrel"
	"github.com/google/uuid"
	"pinnAutomizer/internal/adapter/postgres/schema"
	"pinnAutomizer/internal/domain"
)

func (r *Repository) GetTasksByIDs(
	ctx context.Context,
	ids []uuid.UUID,
	userID uuid.UUID,
) ([]domain.Task, error) {
	query := r.sb.
		Select(schema.TasksTableColumns...).
		From(schema.TasksTable).
		Where(sq.Eq{schema.TasksTableColumnUserId: userID}).
		Where(sq.Eq{schema.TasksTableColumnID: ids})

	var rows []TaskRow
	if err := r.pool.Selectx(ctx, &rows, query); err != nil {
		return nil, err
	}

	tasks := make([]domain.Task, len(rows))
	for i, row := range rows {
		tasks[i] = ToModel(&row)
	}
	return tasks, nil
}
