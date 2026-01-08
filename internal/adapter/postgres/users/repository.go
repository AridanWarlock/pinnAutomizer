package users

import (
	"errors"
	"pinnAutomizer/internal/adapter/postgres/pool"

	"github.com/Masterminds/squirrel"
)

var ErrNotFound = errors.New("not found")

type UsersRepository struct {
	pool pool.Poolx
	sb   squirrel.StatementBuilderType
}

func NewRepository(pool pool.Poolx) *UsersRepository {
	return &UsersRepository{
		pool: pool,
		sb:   squirrel.StatementBuilder.PlaceholderFormat(squirrel.Dollar),
	}
}
