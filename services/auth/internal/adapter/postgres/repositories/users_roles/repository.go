package users_roles

import (
	"github.com/AridanWarlock/pinnAutomizer/pkg/adapter/postgres/poolx"

	"github.com/Masterminds/squirrel"
)

type Repository struct {
	pool poolx.Pool
	sb   squirrel.StatementBuilderType
}

func NewRepository(pool poolx.Pool) *Repository {
	return &Repository{
		pool: pool,
		sb:   squirrel.StatementBuilder.PlaceholderFormat(squirrel.Dollar),
	}
}
