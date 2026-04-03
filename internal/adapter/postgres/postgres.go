package postgres

import (
	"context"
	"fmt"
	"pinnAutomizer/internal/adapter/postgres/pool"
	"pinnAutomizer/internal/adapter/postgres/repositories/equations"
	"pinnAutomizer/internal/adapter/postgres/repositories/events"
	"pinnAutomizer/internal/adapter/postgres/repositories/roles"
	"pinnAutomizer/internal/adapter/postgres/repositories/tasks"
	"pinnAutomizer/internal/adapter/postgres/repositories/user_sessions"
	"pinnAutomizer/internal/adapter/postgres/repositories/users"
	"pinnAutomizer/internal/adapter/postgres/repositories/users_roles"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/rs/zerolog"
)

type Config struct {
	Addr string `env:"POSTGRES_URL" env-required:"true"`
}

type EquationRepository = *equations.Repository
type EventsRepository = *events.Repository
type RolesRepository = *roles.Repository
type TasksRepository = *tasks.Repository
type UsersRepository = *users.Repository
type UsersRolesRepository = *users_roles.Repository
type UserSessionsRepository = *user_sessions.Repository

type Repository struct {
	pool pool.Poolx

	EquationRepository
	EventsRepository
	RolesRepository
	TasksRepository
	UsersRepository
	UsersRolesRepository
	UserSessionsRepository
}

func New(ctx context.Context, c Config, log zerolog.Logger) (*Repository, error) {
	fmt.Println(c.Addr)
	pgxPool, err := pgxpool.New(ctx, c.Addr)
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

		EquationRepository:     equations.NewRepository(poolx),
		EventsRepository:       events.NewRepository(poolx),
		RolesRepository:        roles.NewRepository(poolx),
		TasksRepository:        tasks.NewRepository(poolx),
		UsersRepository:        users.NewRepository(poolx),
		UsersRolesRepository:   users_roles.NewRepository(poolx),
		UserSessionsRepository: user_sessions.NewRepository(poolx),
	}, nil
}

func (r *Repository) Wrap(ctx context.Context, fn func(context.Context) error) error {
	return r.pool.Wrap(ctx, fn)
}

func (r *Repository) Close() {
	r.pool.Close()
}
