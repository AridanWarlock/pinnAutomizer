package users

import (
	"github.com/AridanWarlock/pinnAutomizer/internal/adapter/postgres/pool"

	"github.com/Masterminds/squirrel"
)

type Repository struct {
	pool pool.Pool
	sb   squirrel.StatementBuilderType
}

func NewRepository(pool pool.Pool) *Repository {
	return &Repository{
		pool: pool,
		sb:   squirrel.StatementBuilder.PlaceholderFormat(squirrel.Dollar),
	}
}
