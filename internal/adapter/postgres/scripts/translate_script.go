package scripts

import (
	"context"
	"pinnAutomizer/internal/adapter/postgres/schema"
	"pinnAutomizer/internal/domain"

	"github.com/Masterminds/squirrel"
)

func (r *Repository) TranslateScript(
	ctx context.Context,
	in *domain.FromTranslate,
) error {
	q := r.sb.
		Update(schema.ScriptsTable).
		Set(schema.ScriptsTableColumnText, in.Text).
		Where(squirrel.Eq{schema.ScriptsTableColumnID: in.ScriptID})

	_, err := r.pool.Execx(ctx, q)

	return err
}
