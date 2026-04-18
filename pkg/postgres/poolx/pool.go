package poolx

import (
	"context"
)

type Sqlizer interface {
	ToSql() (sql string, args []any, err error)
}

type Tag interface {
	RowsAffected() int
}

type Pool interface {
	Getx(ctx context.Context, dst any, sqlizer Sqlizer) error
	Selectx(ctx context.Context, dst any, sqlizer Sqlizer) error
	Execx(ctx context.Context, sqlizer Sqlizer) (Tag, error)
	InTransaction(ctx context.Context, inTx func(ctx context.Context) error) error

	Close()
}
