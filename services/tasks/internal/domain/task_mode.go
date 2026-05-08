package domain

import "fmt"

type TaskMode string

const (
	TaskModeTrain   TaskMode = "train"
	TaskModeRetrain TaskMode = "retrain"
	TaskModePredict TaskMode = "predict"
)

func NewTaskMode(mode string) (TaskMode, error) {
	m := TaskMode(mode)
	if err := m.Validate(); err != nil {
		return "", err
	}
	return m, nil
}

func (m TaskMode) RequiredFiles() []string {
	base := []string{"data.mat", "config.yaml", "functions.py"}
	switch m {
	case TaskModeTrain:
		return append(base, "data.mat")
	case TaskModePredict, TaskModeRetrain:
		return append(base, "checkpoint.ckpt")
	default:
		panic(fmt.Sprintf("unexpected task mode: %s", m))
	}
}

func (m TaskMode) Validate() error {
	switch m {
	case TaskModeTrain, TaskModeRetrain, TaskModePredict:
		return nil
	}
	return ErrInvalidTaskMode
}
