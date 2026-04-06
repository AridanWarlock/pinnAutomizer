package equations

import (
	"context"
	"fmt"

	"github.com/AridanWarlock/pinnAutomizer/internal/adapter/postgres/pgerr"
	. "github.com/AridanWarlock/pinnAutomizer/internal/adapter/postgres/schema"
	"github.com/AridanWarlock/pinnAutomizer/internal/domain"
	"github.com/AridanWarlock/pinnAutomizer/internal/errs"

	sq "github.com/Masterminds/squirrel"
)

func (r *Repository) GetEquationByType(ctx context.Context, equationType string) (domain.Equation, error) {
	query := r.sb.
		Select(EquationsTableColumns...).
		From(EquationsTable).
		Where(sq.Eq{EquationsTableColumnType: equationType})

	var row EquationRow
	if err := r.pool.Getx(ctx, &row, query); err != nil {
		if pgerr.IsNotFound(err) {
			return domain.Equation{}, fmt.Errorf(
				"equation by type=%v: %w",
				equationType,
				errs.ErrNotFound,
			)
		}
		return domain.Equation{}, pgerr.ScanErr(err)
	}
	return ToModel(row), nil
}
