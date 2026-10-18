package sqlite

import (
	"database/sql"
	"errors"
	"os"
	"sync"
	"time"

	_ "modernc.org/sqlite"
)

var ErrAlreadyOwned = errors.New("SQLite database already owned")

type SQLite struct {
	timeout  time.Duration
	db       *sql.DB
	owner    *os.File
	once     sync.Once
	closeErr error
}

func (r *SQLite) Close() error {
	r.once.Do(func() {
		if r.db != nil {
			r.closeErr = r.db.Close()
		}
		if r.owner != nil {
			r.closeErr = errors.Join(r.closeErr, r.owner.Close())
		}
	})
	return r.closeErr
}

func (s *SQLite) Database() *sql.DB {
	return s.db
}
