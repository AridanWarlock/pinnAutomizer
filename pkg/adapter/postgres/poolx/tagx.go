package poolx

import "github.com/jackc/pgx/v5/pgconn"

type Tagx struct {
	rowsAffected int
}

func newTagxFromPgx(tag pgconn.CommandTag) Tagx {
	return Tagx{
		rowsAffected: int(tag.RowsAffected()),
	}
}

func (t Tagx) RowsAffected() int {
	return t.rowsAffected
}
