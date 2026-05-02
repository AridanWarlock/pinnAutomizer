package domain

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
	base := []string{"config.yaml", "functions.py"}
	switch m {
	case TaskModeTrain:
		return append(base, "data.mat")
	case TaskModePredict:
		return append(base, "checkpoint.ckpt")
	case TaskModeRetrain:
		return append(base, "data.mat", "checkpoint.ckpt")
	default:
		return nil
	}
}

func (m TaskMode) Validate() error {
	switch m {
	case TaskModeTrain, TaskModeRetrain, TaskModePredict:
		return nil
	}
	return ErrInvalidTaskMode
}
