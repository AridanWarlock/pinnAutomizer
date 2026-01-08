package scripts

import (
	"context"
	"pinnAutomizer/internal/domain"

	"github.com/Masterminds/squirrel"
)

func (r *ScriptsRepository) TranslateScript(
	ctx context.Context,
	in *domain.FromTranslate,
) error {
	q := r.sb.
		Update(scriptsTable).
		Set(scriptsTableColumnText, in.Text).
		Where(squirrel.Eq{scriptsTableColumnID: in.ScriptID})

	_, err := r.pool.Execx(ctx, q)

	return err
}
