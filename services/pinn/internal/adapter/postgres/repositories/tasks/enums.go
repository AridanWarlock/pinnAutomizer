package tasks

import (
	"database/sql/driver"
	"errors"
	"fmt"
)

type TaskStatus string

func (s *TaskStatus) Scan(value any) error {
	if value == nil {
		return errors.New("scan nil value")
	}

	switch v := value.(type) {
	case []byte:
		*s = TaskStatus(v)
	case string:
		*s = TaskStatus(v)
	default:
		return fmt.Errorf("cannot scan %T into TaskStatus", value)
	}
	return nil
}

func (s TaskStatus) Value() (driver.Value, error) {
	return string(s), nil
}
