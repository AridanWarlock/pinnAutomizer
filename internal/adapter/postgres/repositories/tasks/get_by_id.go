package tasks

import (
	"context"
	sq "github.com/Masterminds/squirrel"
	"github.com/google/uuid"
	"pinnAutomizer/internal/adapter/postgres/schema"
	"pinnAutomizer/internal/domain"
)

func (r *Repository) GetTaskByID(ctx context.Context, id uuid.UUID) (domain.Task, error) {
	query := r.sb.
		Select(schema.TasksTableColumns...).
		From(schema.TasksTable).
		Where(sq.Eq{schema.TasksTableColumnID: id})

	var outRow TaskRow
	if err := r.pool.Getx(ctx, &outRow, query); err != nil {
		return domain.Task{}, err
	}

	return ToModel(&outRow), nil
}
