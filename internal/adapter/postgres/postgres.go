package postgres

import (
	"context"
	"fmt"
	"pinnAutomizer/internal/adapter/postgres/pool"
	"pinnAutomizer/internal/adapter/postgres/repositories/auth_tokens"
	"pinnAutomizer/internal/adapter/postgres/repositories/equations"
	"pinnAutomizer/internal/adapter/postgres/repositories/events"
	"pinnAutomizer/internal/adapter/postgres/repositories/roles"
	"pinnAutomizer/internal/adapter/postgres/repositories/tasks"
	"pinnAutomizer/internal/adapter/postgres/repositories/users"
	"pinnAutomizer/internal/adapter/postgres/repositories/users_roles"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/rs/zerolog"
)

type Config struct {
	Addr string `env:"POSTGRES_URL"`
}

type AuthTokensRepository = *auth_tokens.Repository
type EquationRepository = *equations.Repository
type EventsRepository = *events.Repository
type RolesRepository = *roles.Repository
type TasksRepository = *tasks.Repository
type UsersRepository = *users.Repository
type UsersRolesRepository = *users_roles.Repository

type Repository struct {
	pool pool.Poolx

	AuthTokensRepository
	EquationRepository
	EventsRepository
	RolesRepository
	TasksRepository
	UsersRepository
	UsersRolesRepository
}

func New(ctx context.Context, c Config, log zerolog.Logger) (*Repository, error) {
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

	poolx := pool.New(pgxPool, log)

	return &Repository{
		pool: poolx,

		AuthTokensRepository: auth_tokens.NewRepository(poolx),
		EquationRepository:   equations.NewRepository(poolx),
		EventsRepository:     events.NewRepository(poolx),
		RolesRepository:      roles.NewRepository(poolx),
		TasksRepository:      tasks.NewRepository(poolx),
		UsersRepository:      users.NewRepository(poolx),
		UsersRolesRepository: users_roles.NewRepository(poolx),
	}, nil
}

func (r *Repository) Wrap(ctx context.Context, fn func(context.Context) error) error {
	return r.pool.Wrap(ctx, fn)
}

func (r *Repository) Close() {
	r.pool.Close()
}
