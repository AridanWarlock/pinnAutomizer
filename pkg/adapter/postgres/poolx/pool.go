package poolx

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/AridanWarlock/pinnAutomizer/pkg/errs"
	"github.com/AridanWarlock/pinnAutomizer/pkg/logger"
	"github.com/georgysavva/scany/v2/pgxscan"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

type txKey struct{}

type Sqlizer = interface {
	ToSql() (sql string, args []any, err error)
}

func toSqlErr(err error) error {
	return fmt.Errorf("postgres: to sql: %w", err)
}

type executor interface {
	Exec(ctx context.Context, sql string, arguments ...any) (commandTag pgconn.CommandTag, err error)
	Query(ctx context.Context, query string, args ...any) (pgx.Rows, error)
}

type Poolx struct {
	pool *pgxpool.Pool

	timeout time.Duration
}

func New(cfg Config) (*Poolx, error) {
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

	return &Poolx{
		pool:    pgxPool,
		timeout: cfg.Timeout,
	}, nil
}

func (p *Poolx) Getx(ctx context.Context, dst any, sqlizer Sqlizer) error {
	ctx, cancel := context.WithTimeout(ctx, p.timeout)
	defer cancel()

	query, args, err := sqlizer.ToSql()
	if err != nil {
		return toSqlErr(err)
	}

	err = pgxscan.Get(ctx, p.executor(ctx), dst, query, args...)
	if err != nil {
		return p.handleErr(err)
	}
	return nil
}

func (p *Poolx) Selectx(ctx context.Context, dst any, sqlizer Sqlizer) error {
	ctx, cancel := context.WithTimeout(ctx, p.timeout)
	defer cancel()

	query, args, err := sqlizer.ToSql()
	if err != nil {
		return toSqlErr(err)
	}

	err = pgxscan.Select(ctx, p.executor(ctx), dst, query, args...)
	if err != nil {
		return p.handleErr(err)
	}
	return nil
}

func (p *Poolx) Execx(ctx context.Context, sqlizer Sqlizer) (Tag, error) {
	ctx, cancel := context.WithTimeout(ctx, p.timeout)
	defer cancel()

	query, args, err := sqlizer.ToSql()
	if err != nil {
		return Tagx{}, toSqlErr(err)
	}

	tag, err := p.executor(ctx).Exec(ctx, query, args...)
	if err != nil {
		return Tagx{}, p.handleErr(err)
	}
	return newTagxFromPgx(tag), nil
}

func (p *Poolx) InTransaction(ctx context.Context, inTx func(context.Context) error) error {
	ctx, cancel := context.WithTimeout(ctx, p.timeout)
	defer cancel()

	tx, err := p.pool.Begin(ctx)
	if err != nil {
		return err
	}

	defer func() {
		err := tx.Rollback(ctx)
		if err != nil && !errors.Is(err, pgx.ErrTxClosed) {
			log := logger.FromContext(ctx)
			log.Error().Err(err).Msg("poolx: tx.Rollback")
		}
	}()

	ctx = context.WithValue(ctx, txKey{}, tx)

	err = inTx(ctx)
	if err != nil {
		return fmt.Errorf("fn: %w", err)
	}

	err = tx.Commit(ctx)
	if err != nil {
		return fmt.Errorf("tx.Commit: %w", err)
	}
	return nil
}

func (p *Poolx) executor(ctx context.Context) executor {
	tx, ok := ctx.Value(txKey{}).(pgx.Tx)
	if ok {
		return tx
	}
	return p.pool
}

func (p *Poolx) handleErr(err error) error {
	var pgErr *pgconn.PgError

	switch {
	case errors.Is(err, pgx.ErrNoRows):
		return errs.ErrNotFound
	case errors.As(err, &pgErr):
		switch pgErr.Code {
		case "23505", "23503":
			return errs.ErrConflict
		}
	case errs.IsContextErr(err):
		return err
	}

	return fmt.Errorf("postgres err: %w", err)
}

func (p *Poolx) Close() {
	p.pool.Close()
}
