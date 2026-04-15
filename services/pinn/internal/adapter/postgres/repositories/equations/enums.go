package equations

import (
	"database/sql/driver"
	"errors"
	"fmt"
)

type EquationType string

func (t *EquationType) Scan(value any) error {
	if value == nil {
		return errors.New("scan nil value")
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
