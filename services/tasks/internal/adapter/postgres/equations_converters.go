package postgres

import (
	"database/sql/driver"
	"errors"
	"fmt"

	"github.com/AridanWarlock/pinnAutomizer/tasks/internal/domain"
	"github.com/google/uuid"
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

type EquationRow struct {
	ID   uuid.UUID    `db:"id"`
	Type EquationType `db:"type"`
}

func (r *EquationRow) Values() []any {
	return []any{
		r.ID,
		r.Type,
	}
}

func FromEquationModel(e domain.Equation) EquationRow {
	return EquationRow{
		ID:   e.ID,
		Type: EquationType(e.Type),
	}
}

func ToEquationModel(e EquationRow) domain.Equation {
	return domain.Equation{
		ID:   e.ID,
		Type: domain.EquationType(e.Type),
	}
}
