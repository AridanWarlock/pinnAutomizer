package equations

import (
	"context"

	. "github.com/AridanWarlock/pinnAutomizer/internal/adapter/postgres/schema"
	"github.com/AridanWarlock/pinnAutomizer/internal/domain"
	sq "github.com/Masterminds/squirrel"
)

func (r *Repository) GetEquationByType(ctx context.Context, equationType string) (domain.Equation, error) {
	query := r.sb.
		Select(EquationsTableColumns...).
		From(EquationsTable).
		Where(sq.Eq{EquationsTableColumnType: equationType})

	var row EquationRow
	if err := r.pool.Getx(ctx, &row, query); err != nil {
		return domain.Equation{}, err
	}
	return ToModel(row), nil
}
