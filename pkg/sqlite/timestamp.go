package sqlite

import (
	"database/sql/driver"
	"fmt"
	"time"
)

// SQLTime preserves the journal's INTEGER Unix-nanosecond storage format.
type SQLTime struct {
	time.Time
}

func Timestamp(value time.Time) SQLTime {
	return SQLTime{Time: value.UTC()}
}

func NullableTimestamp(value *time.Time) *SQLTime {
	if value == nil {
		return nil
	}
	result := Timestamp(*value)
	return &result
}

func Time(value *SQLTime) *time.Time {
	if value == nil {
		return nil
	}
	result := value.Time
	return &result
}

func (t SQLTime) Value() (driver.Value, error) {
	n := t.UnixNano()
	if !time.Unix(0, n).Equal(t.Time) {
		return nil, fmt.Errorf("SQLite timestamp is outside the Unix nanosecond range")
	}
	return n, nil
}

func (t *SQLTime) Scan(value any) error {
	n, ok := value.(int64)
	if !ok {
		return fmt.Errorf("SQLite timestamp: expected INTEGER, got %T", value)
	}
	t.Time = time.Unix(0, n).UTC()
	return nil
}
