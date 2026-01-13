package equations

import (
	"pinnAutomizer/internal/domain"

	"github.com/google/uuid"
)

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

func FromModel(e domain.Equation) EquationRow {
	return EquationRow{
		ID:   e.ID,
		Type: EquationType(e.Type),
	}
}

func ToModel(e EquationRow) domain.Equation {
	return domain.Equation{
		ID:   e.ID,
		Type: domain.EquationType(e.Type),
	}
}
