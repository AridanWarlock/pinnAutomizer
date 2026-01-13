package equations

import (
	"github.com/Masterminds/squirrel"
	"pinnAutomizer/internal/adapter/postgres/pool"
)

type Repository struct {
	pool pool.Poolx
	sb   squirrel.StatementBuilderType
}

func NewRepository(pool pool.Poolx) *Repository {
	return &Repository{
		pool: pool,
		sb:   squirrel.StatementBuilder.PlaceholderFormat(squirrel.Dollar),
	}
}
