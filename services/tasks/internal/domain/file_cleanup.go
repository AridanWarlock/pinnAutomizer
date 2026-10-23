package domain

import (
	"fmt"
	"path/filepath"

	"github.com/google/uuid"

	"github.com/AridanWarlock/pinnAutomizer/pkg/errs"
	"github.com/AridanWarlock/pinnAutomizer/pkg/validate"
)

// FileCleanup preserves paths for an incomplete creation or deleted task.
// Construct it through NewFileCleanup; callers must not mutate its fields.
type FileCleanup struct {
	TaskID     uuid.UUID `validate:"required,uuid"`
	DataPath   string    `validate:"required"`
	OutputPath string    `validate:"required"`
}

func NewFileCleanup(
	taskID uuid.UUID,
	dataPath string,
	outputPath string,
) (FileCleanup, error) {
	job := FileCleanup{
		TaskID:     taskID,
		DataPath:   dataPath,
		OutputPath: outputPath,
	}
	if err := job.Validate(); err != nil {
		return FileCleanup{}, err
	}
	return job, nil
}

func (j FileCleanup) Validate() error {
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
