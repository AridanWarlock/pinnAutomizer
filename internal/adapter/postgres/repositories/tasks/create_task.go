package tasks

import (
	"context"
	"strings"

	"github.com/AridanWarlock/pinnAutomizer/internal/adapter/postgres/pgerr"
	. "github.com/AridanWarlock/pinnAutomizer/internal/adapter/postgres/schema"
	"github.com/AridanWarlock/pinnAutomizer/internal/domain"
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
		return domain.Task{}, pgerr.ScanErr(err)
	}
	return ToModel(outRow), nil
}
