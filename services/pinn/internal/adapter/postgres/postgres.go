package postgres

import (
	"context"
	"fmt"

	"github.com/AridanWarlock/pinnAutomizer/pinn/internal/adapter/postgres/pool"
	"github.com/AridanWarlock/pinnAutomizer/pinn/internal/adapter/postgres/repositories/equations"
	"github.com/AridanWarlock/pinnAutomizer/pinn/internal/adapter/postgres/repositories/events"
	"github.com/AridanWarlock/pinnAutomizer/pinn/internal/adapter/postgres/repositories/refresh_tokens"
	"github.com/AridanWarlock/pinnAutomizer/pinn/internal/adapter/postgres/repositories/roles"
	"github.com/AridanWarlock/pinnAutomizer/pinn/internal/adapter/postgres/repositories/tasks"
	"github.com/AridanWarlock/pinnAutomizer/pinn/internal/adapter/postgres/repositories/users"
	"github.com/AridanWarlock/pinnAutomizer/pinn/internal/adapter/postgres/repositories/users_roles"
	"github.com/AridanWarlock/pinnAutomizer/pkg/adapter/postgres/poolx"
)

type EquationRepository = *equations.Repository
type EventsRepository = *events.Repository
type RefreshTokensRepository = *refresh_tokens.Repository
type RolesRepository = *roles.Repository
type TasksRepository = *tasks.Repository
type UsersRepository = *users.Repository
type UsersRolesRepository = *users_roles.Repository

type Repository struct {
	pool pool.Pool

	EquationRepository
	EventsRepository
	RefreshTokensRepository
	RolesRepository
	TasksRepository
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

		EquationRepository:      equations.NewRepository(p),
		EventsRepository:        events.NewRepository(p),
		RefreshTokensRepository: refresh_tokens.NewRepository(p),
		RolesRepository:         roles.NewRepository(p),
		TasksRepository:         tasks.NewRepository(p),
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
