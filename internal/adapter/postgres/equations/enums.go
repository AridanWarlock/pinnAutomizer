package equations

import (
	"database/sql/driver"
	"errors"
	"fmt"
)

type EquationType string

var ErrNilScanValue = errors.New("nil scan value into EquationType")

func (t *EquationType) Scan(value any) error {
	if value == nil {
		return ErrNilScanValue
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
