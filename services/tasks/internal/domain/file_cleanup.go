package domain

import (
	"fmt"
	"path/filepath"

	"github.com/google/uuid"

	"github.com/AridanWarlock/pinnAutomizer/pkg/errs"
	"github.com/AridanWarlock/pinnAutomizer/pkg/validate"
)

type fileCleanupData struct {
	TaskID     uuid.UUID `validate:"required,uuid"`
	DataPath   string    `validate:"required"`
	OutputPath string    `validate:"required"`
}

// FileCleanup preserves immutable paths after the task row has been deleted.
type FileCleanup struct {
	data fileCleanupData
}

func NewFileCleanup(
	taskID uuid.UUID,
	dataPath string,
	outputPath string,
) (FileCleanup, error) {
	job := FileCleanup{
		data: fileCleanupData{
			TaskID:     taskID,
			DataPath:   dataPath,
			OutputPath: outputPath,
		},
	}
	if err := job.Validate(); err != nil {
		return FileCleanup{}, err
	}
	return job, nil
}

func (j FileCleanup) Validate() error {
	if err := validate.V.Struct(j.data); err != nil {
		return err
	}
	for _, path := range []string{j.data.DataPath, j.data.OutputPath} {
		if !filepath.IsAbs(path) || filepath.Base(filepath.Clean(path)) != j.data.TaskID.String() {
			return fmt.Errorf("%w: cleanup path must identify its task directory", errs.ErrInvalidArgument)
		}
	}
	if filepath.Clean(j.data.DataPath) == filepath.Clean(j.data.OutputPath) {
		return fmt.Errorf("%w: task directories must be distinct", errs.ErrInvalidArgument)
	}
	return nil
}

func (j FileCleanup) TaskID() uuid.UUID {
	return j.data.TaskID
}

func (j FileCleanup) DataPath() string {
	return j.data.DataPath
}

func (j FileCleanup) OutputPath() string {
	return j.data.OutputPath
}
