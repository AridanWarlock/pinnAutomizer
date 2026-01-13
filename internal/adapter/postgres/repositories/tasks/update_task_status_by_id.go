package tasks

import (
	"context"
	"pinnAutomizer/internal/adapter/postgres/pg_errors"
	"pinnAutomizer/internal/adapter/postgres/schema"

	sq "github.com/Masterminds/squirrel"
	"github.com/google/uuid"
)

func (r *Repository) UpdateTaskStatusByID(ctx context.Context, id uuid.UUID, status string) error {
	query := r.sb.
		Update(schema.TasksTable).
		Set(schema.TasksTableColumnStatus, status).
		Where(sq.Eq{schema.TasksTableColumnID: id})

	tag, err := r.pool.Execx(ctx, query)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return pg_errors.ErrNotFound
	}
	return nil
}
