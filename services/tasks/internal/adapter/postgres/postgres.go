package postgres

import (
	"context"
	"fmt"

	"github.com/AridanWarlock/pinnAutomizer/pkg/adapter/postgres/poolx"
	"github.com/AridanWarlock/pinnAutomizer/tasks/internal/adapter/postgres/repositories/equations"
	"github.com/AridanWarlock/pinnAutomizer/tasks/internal/adapter/postgres/repositories/events"
	"github.com/AridanWarlock/pinnAutomizer/tasks/internal/adapter/postgres/repositories/tasks"
)

type EquationRepository = *equations.Repository
type EventsRepository = *events.Repository
type TasksRepository = *tasks.Repository

type Repository struct {
	pool poolx.Pool

	EquationRepository
	EventsRepository
	TasksRepository
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

		EquationRepository: equations.NewRepository(p),
		EventsRepository:   events.NewRepository(p),
		TasksRepository:    tasks.NewRepository(p),
	}, nil
}

func (r *Repository) InTransaction(ctx context.Context, inTx func(ctx context.Context) error) error {
	return r.pool.InTransaction(ctx, inTx)
}

func (r *Repository) Close() {
	r.pool.Close()
}
