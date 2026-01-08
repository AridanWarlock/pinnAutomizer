package scripts

import (
	"context"
	"pinnAutomizer/internal/adapter/postgres/schema"
	"pinnAutomizer/internal/domain"
	"strings"
)

func (r *Repository) CreateScript(ctx context.Context, in *domain.Script) (*domain.Script, error) {
	row := FromModel(in)

	query := r.sb.
		Insert(schema.ScriptsTable).
		Columns(schema.ScriptsTableColumns...).
		Values(row.Values()...).
		Suffix("RETURNING " + strings.Join(schema.ScriptsTableColumns, ","))

	var outRow ScriptRow
	if err := r.pool.Getx(ctx, &outRow, query); err != nil {
		return nil, err
	}
	return ToModel(&outRow), nil
}
