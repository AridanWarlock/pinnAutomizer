package users

import (
	"errors"
	"pinnAutomizer/internal/adapter/postgres/pool"

	"github.com/Masterminds/squirrel"
)

var ErrNotFound = errors.New("not found")

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
