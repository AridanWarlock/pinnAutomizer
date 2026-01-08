package auth_tokens

import (
	"errors"
	"pinnAutomizer/internal/adapter/postgres/pool"

	"github.com/Masterminds/squirrel"
)

var ErrNotFound = errors.New("not found")

type AuthTokensRepository struct {
	pool pool.Poolx
	sb   squirrel.StatementBuilderType
}

func NewRepository(pool pool.Poolx) *AuthTokensRepository {
	return &AuthTokensRepository{
		pool: pool,
		sb:   squirrel.StatementBuilder.PlaceholderFormat(squirrel.Dollar),
	}
}
