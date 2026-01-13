package tasks

import (
	"database/sql/driver"
	"fmt"
	"pinnAutomizer/internal/adapter/postgres/pg_errors"
)

type TaskStatus string

func (s *TaskStatus) Scan(value any) error {
	if value == nil {
		return pg_errors.ErrNilScanValue
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
