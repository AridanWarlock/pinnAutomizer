package postgres

import (
	"fmt"

	"github.com/AridanWarlock/pinnAutomizer/tasks/internal/domain"
)

func newFileCleanupRow(job domain.FileCleanup) fileCleanupRow {
	return fileCleanupRow{
		TaskID:     job.TaskID(),
		DataPath:   job.DataPath(),
		OutputPath: job.OutputPath(),
	}
}

func (r fileCleanupRow) toDomain() domain.FileCleanup {
	job, err := domain.NewFileCleanup(r.TaskID, r.DataPath, r.OutputPath)
	if err != nil {
		panic(fmt.Errorf("postgres: invalid file cleanup row: %w", err))
	}
	return job
}
