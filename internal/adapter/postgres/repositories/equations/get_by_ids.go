package equations

import (
	"context"
	. "pinnAutomizer/internal/adapter/postgres/schema"
	"pinnAutomizer/internal/domain"

	. "github.com/Masterminds/squirrel"
	"github.com/google/uuid"
)

func (r *Repository) GetEquationsByIDs(ctx context.Context, ids []uuid.UUID) ([]domain.Equation, error) {
	query := r.sb.
		Select(EquationsTableColumns...).
		From(EquationsTable).
		Where(Eq{EquationsTableColumnID: ids})

	var outRows []EquationRow
	if err := r.pool.Selectx(ctx, &outRows, query); err != nil {
		return nil, err
	}

	equations := make([]domain.Equation, len(outRows))
	for i, row := range outRows {
		equations[i] = ToModel(row)
	}

	return equations, nil
}
