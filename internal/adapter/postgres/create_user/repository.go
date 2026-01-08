package create_user

import (
	"errors"
	"pinnAutomizer/internal/adapter/postgres/pool"

	"github.com/Masterminds/squirrel"
)

var ErrNotFound = errors.New("not found")

type CreateUserRepository struct {
	pool pool.Poolx
	sb   squirrel.StatementBuilderType
}

func NewRepository(pool pool.Poolx) *CreateUserRepository {
	return &CreateUserRepository{
		pool: pool,
		sb:   squirrel.StatementBuilder.PlaceholderFormat(squirrel.Dollar),
	}
}
