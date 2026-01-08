package scripts

import (
	"context"
	"pinnAutomizer/internal/domain"
	"strings"
)

func (r *ScriptsRepository) CreateScript(ctx context.Context, in *domain.Script) (*domain.Script, error) {
	row := FromModel(in)

	query := r.sb.
		Insert(scriptsTable).
		Columns(scriptsTableColumns...).
		Values(row.Values()...).
		Suffix("RETURNING " + strings.Join(scriptsTableColumns, ","))

	var outRow ScriptRow
	if err := r.pool.Getx(ctx, &outRow, query); err != nil {
		return nil, err
	}
	return ToModel(&outRow), nil
}
