package scripts

import (
	"errors"
	"pinnAutomizer/internal/adapter/postgres/pool"

	"github.com/Masterminds/squirrel"
)

var ErrNotFound = errors.New("not found")

type ScriptsRepository struct {
	pool pool.Poolx
	sb   squirrel.StatementBuilderType
}

func NewRepository(pool pool.Poolx) *ScriptsRepository {
	return &ScriptsRepository{
		pool: pool,
		sb:   squirrel.StatementBuilder.PlaceholderFormat(squirrel.Dollar),
	}
}
