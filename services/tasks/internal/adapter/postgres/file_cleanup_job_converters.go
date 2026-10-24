package postgres

import (
	"fmt"

	"github.com/google/uuid"

	"github.com/AridanWarlock/pinnAutomizer/tasks/internal/domain"
)

type fileCleanupJobRow struct {
	TaskID     uuid.UUID `db:"task_id"`
	DataPath   string    `db:"data_path"`
	OutputPath string    `db:"output_path"`
}

func (r fileCleanupJobRow) Values() []any {
	return []any{
		r.TaskID,
		r.DataPath,
		r.OutputPath,
	}
}

func newFileCleanupJobRow(job domain.FileCleanupJob) fileCleanupJobRow {
	return fileCleanupJobRow{
		TaskID:     job.TaskID,
		DataPath:   job.DataPath,
		OutputPath: job.OutputPath,
	}
}

func (r fileCleanupJobRow) toDomain() domain.FileCleanupJob {
	job, err := domain.NewFileCleanupJob(r.TaskID, r.DataPath, r.OutputPath)
	if err != nil {
		panic(fmt.Errorf("postgres: invalid file cleanup row: %w", err))
	}
	return job
}
