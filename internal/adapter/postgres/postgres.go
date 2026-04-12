package postgres

import (
	"context"
	"fmt"
	"time"

	"github.com/AridanWarlock/pinnAutomizer/internal/adapter/postgres/pool"
	"github.com/AridanWarlock/pinnAutomizer/internal/adapter/postgres/repositories/equations"
	"github.com/AridanWarlock/pinnAutomizer/internal/adapter/postgres/repositories/events"
	"github.com/AridanWarlock/pinnAutomizer/internal/adapter/postgres/repositories/refresh_tokens"
	"github.com/AridanWarlock/pinnAutomizer/internal/adapter/postgres/repositories/roles"
	"github.com/AridanWarlock/pinnAutomizer/internal/adapter/postgres/repositories/tasks"
	"github.com/AridanWarlock/pinnAutomizer/internal/adapter/postgres/repositories/users"
	"github.com/AridanWarlock/pinnAutomizer/internal/adapter/postgres/repositories/users_roles"
	"github.com/jackc/pgx/v5/pgxpool"
)

type EquationRepository = *equations.Repository
type EventsRepository = *events.Repository
type RefreshTokensRepository = *refresh_tokens.Repository
type RolesRepository = *roles.Repository
type TasksRepository = *tasks.Repository
type UsersRepository = *users.Repository
type UsersRolesRepository = *users_roles.Repository

type Repository struct {
	pool pool.Poolx

	EquationRepository
	EventsRepository
	RefreshTokensRepository
	RolesRepository
	TasksRepository
	UsersRepository
	UsersRolesRepository
}

func New(cfg Config) (*Repository, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	connString := fmt.Sprintf(
		"postgres://%s:%s@%s:%d/%s?sslmode=%s",
		cfg.User,
		cfg.Password,
		cfg.Host,
		cfg.Port,
		cfg.DB,
		cfg.SslMode,
	)
	config, err := pgxpool.ParseConfig(connString)
	if err != nil {
		return nil, fmt.Errorf("parse pgxconfig: %w", err)
	}

	pgxPool, err := pgxpool.NewWithConfig(ctx, config)
	if err != nil {
		return nil, fmt.Errorf("pgxpool connect: %w", err)
	}

	err = pgxPool.Ping(ctx)
	if err != nil {
		return nil, fmt.Errorf("ping postgres: %w", err)
	}

	poolx := pool.New(pgxPool, cfg.Timeout)

	return &Repository{
		pool: poolx,

		EquationRepository:      equations.NewRepository(poolx),
		EventsRepository:        events.NewRepository(poolx),
		RefreshTokensRepository: refresh_tokens.NewRepository(poolx),
		RolesRepository:         roles.NewRepository(poolx),
		TasksRepository:         tasks.NewRepository(poolx),
		UsersRepository:         users.NewRepository(poolx),
		UsersRolesRepository:    users_roles.NewRepository(poolx),
	}, nil
}

func (r *Repository) Wrap(ctx context.Context, fn func(context.Context) error) error {
	return r.pool.Wrap(ctx, fn)
}

func (r *Repository) Close() {
	r.pool.Close()
}
