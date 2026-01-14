package postgres

import (
	"context"
	"fmt"
	"pinnAutomizer/internal/adapter/postgres/pool"
	"pinnAutomizer/internal/adapter/postgres/repositories/auth_tokens"
	"pinnAutomizer/internal/adapter/postgres/repositories/create_user"
	"pinnAutomizer/internal/adapter/postgres/repositories/equations"
	"pinnAutomizer/internal/adapter/postgres/repositories/events"
	"pinnAutomizer/internal/adapter/postgres/repositories/roles"
	"pinnAutomizer/internal/adapter/postgres/repositories/tasks"
	"pinnAutomizer/internal/adapter/postgres/repositories/users"

	"github.com/jackc/pgx/v5/pgxpool"
)

type Config struct {
	Addr string `env:"POSTGRES_URL"`
}

type AuthTokensRepository = *auth_tokens.Repository
type CreateUserRepository = *create_user.Repository
type EquationRepository = *equations.Repository
type EventsRepository = *events.Repository
type RolesRepository = *roles.Repository
type TasksRepository = *tasks.Repository
type UsersRepository = *users.Repository

type Repository struct {
	pool pool.Poolx

	AuthTokensRepository
	CreateUserRepository
	EquationRepository
	EventsRepository
	RolesRepository
	TasksRepository
	UsersRepository
}

func New(ctx context.Context, c Config) (*Repository, error) {
	cfg, err := pgxpool.ParseConfig(c.Addr)
	if err != nil {
		return nil, fmt.Errorf("parse config: %w", err)
	}

	pgxPool, err := pgxpool.NewWithConfig(ctx, cfg)
	if err != nil {
		return nil, fmt.Errorf("pgxpool connect: %w", err)
	}

	err = pgxPool.Ping(ctx)
	if err != nil {
		return nil, err
	}

	poolx := pool.Poolx{Pool: pgxPool}

	return &Repository{
		pool: poolx,

		AuthTokensRepository: auth_tokens.NewRepository(poolx),
		CreateUserRepository: create_user.NewRepository(poolx),
		EquationRepository:   equations.NewRepository(poolx),
		EventsRepository:     events.NewRepository(poolx),
		RolesRepository:      roles.NewRepository(poolx),
		TasksRepository:      tasks.NewRepository(poolx),
		UsersRepository:      users.NewRepository(poolx),
	}, nil
}

func (r *Repository) Wrap(ctx context.Context, fn func(context.Context) error) error {
	return r.pool.Wrap(ctx, fn)
}

func (r *Repository) Close() {
	r.pool.Close()
}
