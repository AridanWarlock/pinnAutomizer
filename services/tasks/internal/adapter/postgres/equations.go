package postgres

import (
	"context"

	"github.com/AridanWarlock/pinnAutomizer/tasks/internal/domain"
	sq "github.com/Masterminds/squirrel"
	"github.com/google/uuid"
)

func (r *Repository) GetEquationsByIDs(ctx context.Context, ids []uuid.UUID) ([]domain.Equation, error) {
	query := r.sb.
		Select(EquationsColumns...).
		From(EquationsTable).
		Where(sq.Eq{EquationsID: ids})

	var outRows []EquationRow
	if err := r.pool.Selectx(ctx, &outRows, query); err != nil {
		return nil, err
	}

	equations := make([]domain.Equation, len(outRows))
	for i, row := range outRows {
		equations[i] = ToEquationModel(row)
	}

	return equations, nil
}

func (r *Repository) GetEquationByType(ctx context.Context, equationType string) (domain.Equation, error) {
	query := r.sb.
		Select(EquationsColumns...).
		From(EquationsTable).
		Where(sq.Eq{EquationsType: equationType})

	var row EquationRow
	if err := r.pool.Getx(ctx, &row, query); err != nil {
		return domain.Equation{}, err
	}
	return ToEquationModel(row), nil
}
