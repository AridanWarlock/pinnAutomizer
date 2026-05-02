package domain

import (
	"fmt"

	"github.com/AridanWarlock/pinnAutomizer/pkg/validate"
	"github.com/google/uuid"
)

type MlTaskMode string

const (
	MlTaskModeTrain   = "train"
	MlTaskModeRetrain = "retrain"
	MlTaskModePredict = "predict"
)

type MlTask struct {
	TaskID uuid.UUID `validate:"required,uuid" json:"task_id"`
	Mode   string    `validate:"required,oneof=train retrain predict" json:"mode"`

	CheckpointFile string `json:"checkpoint_file"`
}

func NewMlTask(
	taskID uuid.UUID,
	mode string,
	checkpointFile *string,
) (MlTask, error) {
	t := MlTask{
		TaskID: taskID,
		Mode:   mode,
	}
	if checkpointFile != nil {
		t.CheckpointFile = *checkpointFile
	}

	if err := t.Validate(); err != nil {
		return MlTask{}, err
	}
	return t, nil
}

func (t MlTask) Validate() error {
	err := validate.V.Struct(t)

	if err != nil {
		return fmt.Errorf("%w: %v", ErrInvalidMLTask, err)
	}
	return nil
}
