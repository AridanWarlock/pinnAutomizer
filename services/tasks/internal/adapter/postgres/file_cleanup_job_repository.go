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

func (r *Repository) QueueFileCleanupJob(ctx context.Context, job domain.FileCleanupJob) error {
	query := r.sb.Insert(FileCleanupJobsTable).
		Columns(fileCleanupJobColumns...).
		Values(newFileCleanupJobRow(job).Values()...)
	_, err := r.pool.Execx(ctx, query)
	return err
}

func (r *Repository) GetFileCleanupJobs(
	ctx context.Context,
	afterTaskID uuid.UUID,
	batch limit.Limit,
) ([]domain.FileCleanupJob, error) {
	query := r.sb.Select(fileCleanupJobColumns...).
		From(FileCleanupJobsTable).
		OrderBy(FileCleanupJobTaskID).
		Limit(batch.Uint64())
	if afterTaskID != uuid.Nil {
		query = query.Where(sq.Gt{FileCleanupJobTaskID: afterTaskID})
	}
	var rows []fileCleanupJobRow
	if err := r.pool.Selectx(ctx, &rows, query); err != nil {
		return nil, err
	}
	jobs := make([]domain.FileCleanupJob, 0, len(rows))
	for _, row := range rows {
		jobs = append(jobs, row.toDomain())
	}
	return jobs, nil
}

func (r *Repository) DeleteFileCleanupJob(ctx context.Context, taskID uuid.UUID) error {
	query := r.sb.Delete(FileCleanupJobsTable).
		Where(sq.Eq{FileCleanupJobTaskID: taskID})
	_, err := r.pool.Execx(ctx, query)
	return err
}

// HasFileCleanupJob is called outside an explicit transaction. NOWAIT defers
// cleanup on an in-flight commit; an acquired lock lasts only this statement.
func (r *Repository) HasFileCleanupJob(ctx context.Context, taskID uuid.UUID) (bool, error) {
	// A plain MVCC read could see the old cleanup row during creation commit.
	// A locked row must be retried later, never treated as permission to remove files.
	query := r.sb.Select("true").
		From(FileCleanupJobsTable).
		Where(sq.Eq{FileCleanupJobTaskID: taskID}).
		Suffix("FOR UPDATE NOWAIT")
	var pending bool
	err := r.pool.Getx(ctx, &pending, query)
	if errors.Is(err, errs.ErrNotFound) {
		return false, nil
	}
	return pending, err
}
