package postgres

import (
	"context"
	"fmt"
	"pinnAutomizer/internal/adapter/postgres/auth_tokens"
	"pinnAutomizer/internal/adapter/postgres/create_user"
	"pinnAutomizer/internal/adapter/postgres/equations"
	"pinnAutomizer/internal/adapter/postgres/pool"
	"pinnAutomizer/internal/adapter/postgres/scripts"
	"pinnAutomizer/internal/adapter/postgres/tasks"
	"pinnAutomizer/internal/adapter/postgres/users"

	"github.com/jackc/pgx/v5/pgxpool"
)

type Config struct {
	User     string `env:"POSTGRES_USER" required:"true"`
	Password string `env:"POSTGRES_PASSWORD" required:"true"`
	Host     string `env:"POSTGRES_HOST" required:"true"`
	Port     string `env:"POSTGRES_PORT" required:"true"`
	DBName   string `env:"POSTGRES_DB_NAME" required:"true"`
}

type CreateUserRepository = *create_user.Repository
type AuthTokensRepository = *auth_tokens.Repository
type EquationRepository = *equations.Repository
type ScriptsRepository = *scripts.Repository
type TasksRepository = *tasks.Repository
type UsersRepository = *users.Repository

type Repository struct {
	pool pool.Poolx

	CreateUserRepository
	AuthTokensRepository
	EquationRepository
	TasksRepository
	UsersRepository
	ScriptsRepository
}

func New(ctx context.Context, c Config) (*Repository, error) {
	dsn := fmt.Sprintf("postgres://%s:%s@%s:%s/%s?sslmode=disable",
		c.User, c.Password, c.Host, c.Port, c.DBName,
	)
	cfg, err := pgxpool.ParseConfig(dsn)
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

		CreateUserRepository: create_user.NewRepository(poolx),
		AuthTokensRepository: auth_tokens.NewRepository(poolx),
		EquationRepository:   equations.NewRepository(poolx),
		ScriptsRepository:    scripts.NewRepository(poolx),
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
