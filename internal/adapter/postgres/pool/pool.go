package pool

import (
	"context"
	"errors"
	"fmt"

	"github.com/georgysavva/scany/v2/pgxscan"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/rs/zerolog"
)

type Sqlizer interface {
	ToSql() (sql string, args []interface{}, err error)
}

type Poolx struct {
	*pgxpool.Pool

	log zerolog.Logger
}

func New(pool *pgxpool.Pool, log zerolog.Logger) Poolx {
	return Poolx{
		Pool: pool,
		log:  log.With().Str("component", "poolx").Logger(),
	}
}

type Executor interface {
	Exec(ctx context.Context, sql string, arguments ...any) (pgconn.CommandTag, error)
}

type Querier interface {
	pgxscan.Querier
}

type ExecQuerier interface {
	Executor
	Querier
}

type txKey struct{}

func (p *Poolx) execQuerier(ctx context.Context) ExecQuerier {
	if tx, ok := ctx.Value(txKey{}).(pgx.Tx); ok {
		return tx
	}
	return p.Pool
}

func toSqlErr(err error) error {
	return fmt.Errorf("postgres: to sql: %w", err)
}

func (p *Poolx) Getx(ctx context.Context, dest interface{}, sqlizer Sqlizer) error {
	query, args, err := sqlizer.ToSql()
	if err != nil {
		return toSqlErr(err)
	}

	return pgxscan.Get(ctx, p.execQuerier(ctx), dest, query, args...)
}

func (p *Poolx) Selectx(ctx context.Context, dest interface{}, sqlizer Sqlizer) error {
	query, args, err := sqlizer.ToSql()
	if err != nil {
		return toSqlErr(err)
	}

	return pgxscan.Select(ctx, p.execQuerier(ctx), dest, query, args...)
}

func (p *Poolx) Execx(ctx context.Context, sqlizer Sqlizer) (pgconn.CommandTag, error) {
	query, args, err := sqlizer.ToSql()
	if err != nil {
		return pgconn.CommandTag{}, toSqlErr(err)
	}

	return p.execQuerier(ctx).Exec(ctx, query, args...)
}

func (p *Poolx) Wrap(ctx context.Context, fn func(context.Context) error) error {
	tx, err := p.Pool.Begin(ctx)
	if err != nil {
		return err
	}

	defer func() {
		err := tx.Rollback(ctx)
		if err != nil && !errors.Is(err, pgx.ErrTxClosed) {
			p.log.Error().Err(err).Msg("poolx: tx.Roolback")
		}
	}()

	ctx = context.WithValue(ctx, txKey{}, tx)

	err = fn(ctx)
	if err != nil {
		return fmt.Errorf("fn: %w", err)
	}
	err = tx.Commit(ctx)
	if err != nil {
		return fmt.Errorf("tx.Commit: %w", err)
	}
	return nil
}
