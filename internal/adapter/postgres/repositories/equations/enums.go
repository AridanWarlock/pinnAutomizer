package equations

import (
	"database/sql/driver"
	"fmt"
	"pinnAutomizer/internal/adapter/postgres/pg_errors"
)

type EquationType string

func (t *EquationType) Scan(value any) error {
	if value == nil {
		return pg_errors.ErrNilScanValue
	}

	switch v := value.(type) {
	case []byte:
		*t = EquationType(v)
	case string:
		*t = EquationType(v)
	default:
		return fmt.Errorf("cannot scan %T into EquationType", value)
	}
	return nil
}

func (t EquationType) Value() (driver.Value, error) {
	return string(t), nil
}
