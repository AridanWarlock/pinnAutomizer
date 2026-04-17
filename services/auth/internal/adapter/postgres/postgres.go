package postgres

import (
	"context"
	"fmt"

	"github.com/AridanWarlock/pinnAutomizer/auth/internal/adapter/postgres/repositories/refresh_tokens"
	"github.com/AridanWarlock/pinnAutomizer/auth/internal/adapter/postgres/repositories/roles"
	"github.com/AridanWarlock/pinnAutomizer/auth/internal/adapter/postgres/repositories/users"
	"github.com/AridanWarlock/pinnAutomizer/auth/internal/adapter/postgres/repositories/users_roles"
	"github.com/AridanWarlock/pinnAutomizer/pkg/adapter/postgres/poolx"
)

type RefreshTokensRepository = *refresh_tokens.Repository
type RolesRepository = *roles.Repository
type UsersRepository = *users.Repository
type UsersRolesRepository = *users_roles.Repository

type Repository struct {
	pool poolx.Pool

	RefreshTokensRepository
	RolesRepository
	UsersRepository
	UsersRolesRepository
}

func New(cfg Config) (*Repository, error) {
	p, err := poolx.New(poolx.Config{
		User:     cfg.User,
		Password: cfg.Password,
		Host:     cfg.Host,
		Port:     cfg.Port,
		DB:       cfg.DB,
		SslMode:  cfg.SslMode,
		Timeout:  cfg.Timeout,
	})

	if err != nil {
		return nil, fmt.Errorf("postgres: poolx create: %w", err)
	}

	return &Repository{
		pool: p,

		RefreshTokensRepository: refresh_tokens.NewRepository(p),
		RolesRepository:         roles.NewRepository(p),
		UsersRepository:         users.NewRepository(p),
		UsersRolesRepository:    users_roles.NewRepository(p),
	}, nil
}

func (r *Repository) InTransaction(ctx context.Context, inTx func(ctx context.Context) error) error {
	return r.pool.InTransaction(ctx, inTx)
}

func (r *Repository) Close() {
	r.pool.Close()
}
