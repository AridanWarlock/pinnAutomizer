package postgres

import (
	"context"
	"errors"

	sq "github.com/Masterminds/squirrel"
	"github.com/google/uuid"

	"github.com/AridanWarlock/pinnAutomizer/pkg/core/limit"
	"github.com/AridanWarlock/pinnAutomizer/pkg/errs"
	"github.com/AridanWarlock/pinnAutomizer/tasks/internal/domain"
)

func (r *Repository) QueueFileCleanup(ctx context.Context, job domain.FileCleanup) error {
	query := r.sb.Insert(FileCleanupTable).
		Columns(fileCleanupColumns...).
		Values(newFileCleanupRow(job).Values()...)
	_, err := r.pool.Execx(ctx, query)
	return err
}

func (r *Repository) GetFileCleanups(
	ctx context.Context,
	afterTaskID uuid.UUID,
	batch limit.Limit,
) ([]domain.FileCleanup, error) {
	query := r.sb.Select(fileCleanupColumns...).
		From(FileCleanupTable).
		OrderBy(FileCleanupTaskID).
		Limit(batch.Uint64())
	if afterTaskID != uuid.Nil {
		query = query.Where(sq.Gt{FileCleanupTaskID: afterTaskID})
	}
	var rows []fileCleanupRow
	if err := r.pool.Selectx(ctx, &rows, query); err != nil {
		return nil, err
	}
	jobs := make([]domain.FileCleanup, 0, len(rows))
	for _, row := range rows {
		jobs = append(jobs, row.toDomain())
	}
	return jobs, nil
}

func (r *Repository) DeleteFileCleanup(ctx context.Context, taskID uuid.UUID) error {
	query := r.sb.Delete(FileCleanupTable).
		Where(sq.Eq{FileCleanupTaskID: taskID})
	_, err := r.pool.Execx(ctx, query)
	return err
}

// HasFileCleanup is called outside an explicit transaction. NOWAIT defers
// cleanup on an in-flight commit; an acquired lock lasts only this statement.
func (r *Repository) HasFileCleanup(ctx context.Context, taskID uuid.UUID) (bool, error) {
	// A plain MVCC read could see the old cleanup row during creation commit.
	// A locked row must be retried later, never treated as permission to remove files.
	query := r.sb.Select("true").
		From(FileCleanupTable).
		Where(sq.Eq{FileCleanupTaskID: taskID}).
		Suffix("FOR UPDATE NOWAIT")
	var pending bool
	err := r.pool.Getx(ctx, &pending, query)
	if errors.Is(err, errs.ErrNotFound) {
		return false, nil
	}
	return pending, err
}
