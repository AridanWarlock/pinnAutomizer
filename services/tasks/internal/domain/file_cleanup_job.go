package domain

import (
	"fmt"
	"path/filepath"

	"github.com/google/uuid"

	"github.com/AridanWarlock/pinnAutomizer/pkg/errs"
	"github.com/AridanWarlock/pinnAutomizer/pkg/validate"
)

// FileCleanupJob preserves paths for an incomplete creation or deleted task.
// Construct it through NewFileCleanupJob; callers must not mutate its fields.
type FileCleanupJob struct {
	TaskID     uuid.UUID `validate:"required,uuid"`
	DataPath   string    `validate:"required"`
	OutputPath string    `validate:"required"`
}

func NewFileCleanupJob(
	taskID uuid.UUID,
	dataPath string,
	outputPath string,
) (FileCleanupJob, error) {
	job := FileCleanupJob{
		TaskID:     taskID,
		DataPath:   dataPath,
		OutputPath: outputPath,
	}
	if err := job.Validate(); err != nil {
		return FileCleanupJob{}, err
	}
	return job, nil
}

func (j FileCleanupJob) Validate() error {
	if err := validate.V.Struct(j); err != nil {
		return err
	}
	for _, path := range []string{j.DataPath, j.OutputPath} {
		if !filepath.IsAbs(path) || filepath.Base(filepath.Clean(path)) != j.TaskID.String() {
			return fmt.Errorf("%w: cleanup path must identify its task directory", errs.ErrInvalidArgument)
		}
	}
	if filepath.Clean(j.DataPath) == filepath.Clean(j.OutputPath) {
		return fmt.Errorf("%w: task directories must be distinct", errs.ErrInvalidArgument)
	}
	return nil
}
