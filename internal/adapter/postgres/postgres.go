package postgres

import (
	"context"
	"fmt"
	"pinnAutomizer/internal/adapter/postgres/auth_tokens"
	"pinnAutomizer/internal/adapter/postgres/create_user"
	"pinnAutomizer/internal/adapter/postgres/pool"
	"pinnAutomizer/internal/adapter/postgres/scripts"
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

type Repository struct {
	pool pool.Poolx

	*create_user.CreateUserRepository
	*auth_tokens.AuthTokensRepository
	*scripts.ScriptsRepository
	*users.UsersRepository
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
		ScriptsRepository:    scripts.NewRepository(poolx),
		UsersRepository:      users.NewRepository(poolx),
	}, nil
}

func (r *Repository) Close() {
	r.pool.Close()
}
