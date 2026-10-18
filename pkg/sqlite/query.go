package sqlite

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/georgysavva/scany/v2/sqlscan"
	driver "modernc.org/sqlite"
	sqlite3 "modernc.org/sqlite/lib"

	"github.com/AridanWarlock/pinnAutomizer/pkg/errs"
)

type Sqlizer interface {
	ToSql() (sql string, args []any, err error)
}

type Tag interface {
	RowsAffected() int
}

type tag struct {
	rowsAffected int
}

func (t tag) RowsAffected() int {
	return t.rowsAffected
}

type executor interface {
	ExecContext(context.Context, string, ...any) (sql.Result, error)
	QueryContext(context.Context, string, ...any) (*sql.Rows, error)
}

type transactionKey struct {
	sqlite *SQLite
}

func (s *SQLite) executor(ctx context.Context) executor {
	if tx, ok := ctx.Value(transactionKey{sqlite: s}).(*sql.Tx); ok {
		return tx
	}
	return s.db
}

func (s *SQLite) Getx(ctx context.Context, dst any, sqlizer Sqlizer) error {
	query, args, err := buildQuery(sqlizer)
	if err != nil {
		return err
	}
	ctx, cancel := context.WithTimeout(ctx, s.timeout)
	defer cancel()
	return handleError(sqlscan.Get(ctx, s.executor(ctx), dst, query, args...))
}

func (s *SQLite) Selectx(ctx context.Context, dst any, sqlizer Sqlizer) error {
	query, args, err := buildQuery(sqlizer)
	if err != nil {
		return err
	}
	ctx, cancel := context.WithTimeout(ctx, s.timeout)
	defer cancel()
	return handleError(sqlscan.Select(ctx, s.executor(ctx), dst, query, args...))
}

func (s *SQLite) Execx(ctx context.Context, sqlizer Sqlizer) (Tag, error) {
	query, args, err := buildQuery(sqlizer)
	if err != nil {
		return nil, err
	}
	ctx, cancel := context.WithTimeout(ctx, s.timeout)
	defer cancel()
	result, err := s.executor(ctx).ExecContext(ctx, query, args...)
	if err != nil {
		return nil, handleError(err)
	}
	affected, err := result.RowsAffected()
	if err != nil {
		return nil, err
	}
	return tag{rowsAffected: int(affected)}, nil
}

// InTransaction binds this client's operations to a short database transaction.
func (s *SQLite) InTransaction(ctx context.Context, inTx func(context.Context) error) (err error) {
	if inTx == nil {
		return fmt.Errorf("%w: nil transaction callback", errs.ErrInvalidArgument)
	}
	if _, ok := ctx.Value(transactionKey{sqlite: s}).(*sql.Tx); ok {
		return fmt.Errorf("%w: nested SQLite transaction", errs.ErrInvalidArgument)
	}
	ctx, cancel := context.WithTimeout(ctx, s.timeout)
	defer cancel()
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return handleError(err)
	}
	defer func() {
		rollbackErr := tx.Rollback()
		if !errors.Is(rollbackErr, sql.ErrTxDone) {
			err = errors.Join(err, rollbackErr)
		}
	}()
	ctx = context.WithValue(ctx, transactionKey{sqlite: s}, tx)
	if err = inTx(ctx); err != nil {
		return err
	}
	return handleError(tx.Commit())
}

func buildQuery(sqlizer Sqlizer) (string, []any, error) {
	if sqlizer == nil {
		return "", nil, fmt.Errorf("%w: nil SQL builder", errs.ErrInvalidArgument)
	}
	query, args, err := sqlizer.ToSql()
	if err != nil {
		return "", nil, fmt.Errorf("sqlite: to sql: %w", err)
	}
	return query, args, nil
}

func handleError(err error) error {
	if err == nil {
		return nil
	}
	if sqlscan.NotFound(err) {
		return errs.ErrNotFound
	}
	var sqliteErr *driver.Error
	if errors.As(err, &sqliteErr) {
		switch sqliteErr.Code() {
		case sqlite3.SQLITE_CONSTRAINT_UNIQUE, sqlite3.SQLITE_CONSTRAINT_PRIMARYKEY, sqlite3.SQLITE_CONSTRAINT_FOREIGNKEY:
			return fmt.Errorf("%w: %v", errs.ErrConflict, err)
		}
	}
	return fmt.Errorf("sqlite: %w", err)
}

const defaultTimeout = 5 * time.Second
