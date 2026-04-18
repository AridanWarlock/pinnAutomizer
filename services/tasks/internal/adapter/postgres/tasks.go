package postgres

import (
	"context"
	"fmt"
	"strings"

	"github.com/AridanWarlock/pinnAutomizer/pkg/errs"
	"github.com/AridanWarlock/pinnAutomizer/tasks/internal/domain"
	sq "github.com/Masterminds/squirrel"
	"github.com/google/uuid"
)

func (r *Repository) CreateTask(ctx context.Context, task domain.Task) (domain.Task, error) {
	row := FromTaskModel(task)

	query := r.sb.
		Insert(TasksTable).
		Columns(TasksColumns...).
		Values(row.Values()...).
		Suffix("RETURNING " + strings.Join(TasksColumns, ","))

	var outRow TaskRow
	if err := r.pool.Getx(ctx, &outRow, query); err != nil {
		return domain.Task{}, err
	}
	return ToTaskModel(outRow), nil
}

func (r *Repository) GetTaskByID(ctx context.Context, id uuid.UUID) (domain.Task, error) {
	query := r.sb.
		Select(TasksColumns...).
		From(TasksTable).
		Where(sq.Eq{TasksID: id})

	var outRow TaskRow
	if err := r.pool.Getx(ctx, &outRow, query); err != nil {
		return domain.Task{}, err
	}

	return ToTaskModel(outRow), nil
}

func (r *Repository) GetTasksByIDs(
	ctx context.Context,
	ids []uuid.UUID,
	userID uuid.UUID,
) ([]domain.Task, error) {
	query := r.sb.
		Select(TasksColumns...).
		From(TasksTable).
		Where(sq.Eq{TasksUserId: userID}).
		Where(sq.Eq{TasksID: ids})

	var rows []TaskRow
	if err := r.pool.Selectx(ctx, &rows, query); err != nil {
		return nil, err
	}

	tasks := make([]domain.Task, len(rows))
	for i, row := range rows {
		tasks[i] = ToTaskModel(row)
	}
	return tasks, nil
}

func (r *Repository) GetTaskByIDAndUserID(
	ctx context.Context,
	id uuid.UUID,
	userID uuid.UUID,
) (domain.Task, error) {
	query := r.sb.
		Select(TasksColumns...).
		From(TasksTable).
		Where(sq.Eq{TasksID: id, TasksUserId: userID})

	var outRow TaskRow
	if err := r.pool.Getx(ctx, &outRow, query); err != nil {
		return domain.Task{}, err
	}
	return ToTaskModel(outRow), nil
}

func (r *Repository) UpdateTaskStatusByID(ctx context.Context, id uuid.UUID, status, oldStatus string) error {
	query := r.sb.
		Update(TasksTable).
		Set(TasksStatus, status).
		Where(sq.Eq{TasksID: id, TasksStatus: oldStatus})

	tag, err := r.pool.Execx(ctx, query)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return fmt.Errorf(
			"task with id=%v: %w",
			id,
			errs.ErrNotFound,
		)
	}
	return nil
}
