package postgres

import (
	"fmt"

	"github.com/google/uuid"

	"github.com/AridanWarlock/pinnAutomizer/tasks/internal/domain"
)

type fileCleanupRow struct {
	TaskID     uuid.UUID `db:"task_id"`
	DataPath   string    `db:"data_path"`
	OutputPath string    `db:"output_path"`
}

func (r fileCleanupRow) Values() []any {
	return []any{
		r.TaskID,
		r.DataPath,
		r.OutputPath,
	}
}

func newFileCleanupRow(job domain.FileCleanup) fileCleanupRow {
	return fileCleanupRow{
		TaskID:     job.TaskID,
		DataPath:   job.DataPath,
		OutputPath: job.OutputPath,
	}
}

func (r fileCleanupRow) toDomain() domain.FileCleanup {
	job, err := domain.NewFileCleanup(r.TaskID, r.DataPath, r.OutputPath)
	if err != nil {
		panic(fmt.Errorf("postgres: invalid file cleanup row: %w", err))
	}
	return job
}
