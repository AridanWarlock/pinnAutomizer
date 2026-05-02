package postgres

import (
	"context"
	"fmt"
	"strings"

	"github.com/AridanWarlock/pinnAutomizer/pkg/core/pagination"
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

func (r *Repository) GetTasksByUserID(
	ctx context.Context,
	userID uuid.UUID,
	opts pagination.Options,
) ([]domain.Task, error) {
	q := r.sb.Select(TasksColumns...).
		From(TasksTable).
		Where(sq.Eq{TasksUserId: userID})

	if limit := opts.Limit(); limit != nil {
		q = q.Limit(uint64(*limit))
	}
	if offset := opts.Offset(); offset != nil {
		q = q.Offset(uint64(*offset))
	}
	var orderBys []string
	for _, sf := range opts.OrderBy() {
		if _, ok := TasksSortColumns[sf.Name]; ok {
			orderBys = append(orderBys, sf.String())
		}
	}
	if len(orderBys) == 0 {
		orderBys = append(orderBys, TasksCreatedAt+" DESC")
	}
	q = q.OrderBy(orderBys...)

	var rows []TaskRow
	if err := r.pool.Selectx(ctx, &rows, q); err != nil {
		return nil, err
	}

	tasks := make([]domain.Task, len(rows))
	for i, row := range rows {
		tasks[i] = ToTaskModel(row)
	}
	return tasks, nil
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

func (r *Repository) UpdateTaskStatusByID(ctx context.Context, id uuid.UUID, status domain.TaskStatus) error {
	query := r.sb.
		Update(TasksTable).
		Set(TasksStatus, status).
		Where(sq.Eq{TasksID: id})

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

func (r *Repository) UpdateTaskStatusAndErrorByID(ctx context.Context, id uuid.UUID, status domain.TaskStatus, errorMsg string) error {
	query := r.sb.
		Update(TasksTable).
		Set(TasksStatus, status).
		Set(TasksError, errorMsg).
		Where(sq.Eq{TasksID: id})

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
