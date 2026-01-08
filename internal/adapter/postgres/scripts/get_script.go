package scripts

import (
	"context"
	"errors"
	"pinnAutomizer/internal/adapter/postgres/schema"
	"pinnAutomizer/internal/domain"

	"github.com/Masterminds/squirrel"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

func (r *Repository) GetScript(ctx context.Context, id uuid.UUID) (*domain.Script, error) {
	query := r.sb.
		Select(schema.ScriptsTableColumns...).
		From(schema.ScriptsTable).
		Where(squirrel.Eq{schema.ScriptsTableColumnID: id})

	var row ScriptRow
	if err := r.pool.Getx(ctx, &row, query); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, err
	}

	return ToModel(&row), nil
}
