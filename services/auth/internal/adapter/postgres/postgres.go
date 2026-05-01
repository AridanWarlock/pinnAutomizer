package postgres

import (
	"context"
	"fmt"
	"time"

	"github.com/AridanWarlock/pinnAutomizer/pkg/postgres/poolx"
	sq "github.com/Masterminds/squirrel"
)

type Config struct {
	User     string        `env:"USER,required"`
	Password string        `env:"PASSWORD,required"`
	Host     string        `env:"HOST,required"`
	Port     int           `env:"PORT,required"`
	DB       string        `env:"DB,required"`
	SslMode  string        `env:"SSLMODE" default:"disable"`
	Timeout  time.Duration `env:"TIMEOUT,required"`
}

type Repository struct {
	pool poolx.Pool
	sb   sq.StatementBuilderType
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
		sb:   sq.StatementBuilder.PlaceholderFormat(sq.Dollar),
	}, nil
}

func (r *Repository) InTransaction(ctx context.Context, inTx func(ctx context.Context) error) error {
	return r.pool.InTransaction(ctx, inTx)
}

func (r *Repository) Close() {
	r.pool.Close()
}
