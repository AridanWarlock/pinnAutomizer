package poolx

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/georgysavva/scany/v2/pgxscan"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/AridanWarlock/pinnAutomizer/pkg/errs"
)

type TxManager interface {
	InTransaction(
		ctx context.Context,
		inTx func(context.Context) error,
	) error
}

type txKey struct {
	postgres *Poolx
}

func toSqlErr(err error) error {
	return fmt.Errorf("postgres: to sql: %w", err)
}

type executor interface {
	Exec(ctx context.Context, sql string, arguments ...any) (commandTag pgconn.CommandTag, err error)
	Query(ctx context.Context, query string, args ...any) (pgx.Rows, error)
}

type Postgres interface {
	executor
	Begin(ctx context.Context) (pgx.Tx, error)
	Ping(ctx context.Context) error
	Close()
}

type Poolx struct {
	postgres Postgres

	timeout time.Duration
}

func New(cfg Config) (Pool, error) {
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

	return newPoolx(ctx, pgxPool, cfg.Timeout)
}

// newPoolx takes ownership: failed verification closes the pool, success transfers
// it to the returned client, whose owner must call Close.
func newPoolx(
	ctx context.Context,
	postgres Postgres,
	timeout time.Duration,
) (client Pool, err error) {
	defer func() {
		if err != nil {
			postgres.Close()
		}
	}()
	if err := postgres.Ping(ctx); err != nil {
		return nil, fmt.Errorf("ping postgres: %w", err)
	}
	return &Poolx{
		postgres: postgres,
		timeout:  timeout,
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

func (p *Poolx) InTransaction(
	ctx context.Context,
	inTx func(context.Context) error,
) (err error) {
	if inTx == nil {
		return fmt.Errorf("%w: nil transaction callback", errs.ErrInvalidArgument)
	}
	if _, ok := ctx.Value(txKey{postgres: p}).(pgx.Tx); ok {
		return fmt.Errorf("%w: nested PostgreSQL transaction", errs.ErrInvalidArgument)
	}
	ctx, cancel := context.WithTimeout(ctx, p.timeout)
	defer cancel()

	tx, err := p.postgres.Begin(ctx)
	if err != nil {
		return err
	}

	defer func() {
		rollbackErr := tx.Rollback(ctx)
		if rollbackErr != nil && !errors.Is(rollbackErr, pgx.ErrTxClosed) {
			err = errors.Join(err, fmt.Errorf("tx.Rollback: %w", rollbackErr))
		}
	}()

	ctx = context.WithValue(ctx, txKey{postgres: p}, tx)

	err = inTx(ctx)
	if err != nil {
		return err
	}

	err = tx.Commit(ctx)
	if err != nil {
		return fmt.Errorf("tx.Commit: %w", err)
	}
	return nil
}

func (p *Poolx) executor(ctx context.Context) executor {
	tx, ok := ctx.Value(txKey{postgres: p}).(pgx.Tx)
	if ok {
		return tx
	}
	return p.postgres
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
	p.postgres.Close()
}
