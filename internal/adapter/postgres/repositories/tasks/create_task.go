package tasks

import (
	"context"
	. "pinnAutomizer/internal/adapter/postgres/schema"
	"pinnAutomizer/internal/domain"
	"strings"
)

func (r *Repository) CreateTask(ctx context.Context, task domain.Task) (domain.Task, error) {
	row := FromModel(task)

	query := r.sb.
		Insert(TasksTable).
		Columns(TasksTableColumns...).
		Values(row.Values()...).
		Suffix("RETURNING " + strings.Join(TasksTableColumns, ","))

	var outRow TaskRow
	if err := r.pool.Getx(ctx, &outRow, query); err != nil {
		return domain.Task{}, err
	}
	return ToModel(outRow), nil
}
