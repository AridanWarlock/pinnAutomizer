package tasks

import (
	"context"

	. "github.com/AridanWarlock/pinnAutomizer/internal/adapter/postgres/pg_errors"
	. "github.com/AridanWarlock/pinnAutomizer/internal/adapter/postgres/schema"

	. "github.com/Masterminds/squirrel"
	"github.com/google/uuid"
)

func (r *Repository) UpdateTaskStatusByID(ctx context.Context, id uuid.UUID, status, oldStatus string) error {
	query := r.sb.
		Update(TasksTable).
		Set(TasksTableColumnStatus, status).
		Where(Eq{TasksTableColumnID: id, TasksTableColumnStatus: oldStatus})

	tag, err := r.pool.Execx(ctx, query)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}
