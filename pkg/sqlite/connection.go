package sqlite

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"strconv"
	"time"

	"github.com/AridanWarlock/pinnAutomizer/pkg/errs"
)

func New(cfg Config) (result *SQLite, err error) {
	if cfg.Path == "" || cfg.Path == ":memory:" || cfg.BusyTimeout < time.Millisecond {
		return nil, fmt.Errorf("%w: database path and positive busy timeout required", errs.ErrInvalidArgument)
	}
	path, err := preparePath(cfg.Path)
	if err != nil {
		return nil, err
	}
	owner, err := acquireOwner(path)
	if err != nil {
		return nil, err
	}
	client := &SQLite{owner: owner}
	// Ownership transfers to the caller only after complete initialization.
	defer func() {
		if result == nil {
			err = errors.Join(err, client.Close())
		}
	}()
	if err = checkDatabaseFile(path); err != nil {
		return nil, err
	}
	client.db, err = openDatabase(path, cfg.BusyTimeout)
	if err != nil {
		return nil, err
	}
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err = client.db.PingContext(ctx); err != nil {
		return nil, err
	}
	return client, nil
}

func preparePath(path string) (string, error) {
	path, err := filepath.Abs(path)
	if err != nil {
		return "", err
	}
	if err = os.MkdirAll(filepath.Dir(path), 0700); err != nil {
		return "", err
	}
	dir, err := filepath.EvalSymlinks(filepath.Dir(path))
	if err != nil {
		return "", err
	}
	path = filepath.Join(dir, filepath.Base(path))
	resolved, err := filepath.EvalSymlinks(path)
	if errors.Is(err, os.ErrNotExist) {
		return path, nil
	}
	if err != nil {
		return "", err
	}
	return resolved, nil
}

func acquireOwner(path string) (owner *os.File, err error) {
	file, err := os.OpenFile(path+".lock", os.O_CREATE|os.O_RDWR, 0600)
	if err != nil {
		return nil, err
	}
	defer func() {
		if owner == nil {
			err = errors.Join(err, file.Close())
		}
	}()
	if err = lock(file); err != nil {
		return nil, err
	}
	return file, nil
}

func checkDatabaseFile(path string) (err error) {
	file, err := os.OpenFile(path, os.O_RDWR, 0600)
	if err != nil {
		return err
	}
	defer func() { err = errors.Join(err, file.Close()) }()
	return nil
}

func openDatabase(path string, busyTimeout time.Duration) (*sql.DB, error) {
	uri := url.URL{Scheme: "file", Path: path}
	query := uri.Query()
	query.Set("mode", "rw")
	for _, pragma := range []string{
		"journal_mode(WAL)",
		"synchronous(FULL)",
		"foreign_keys(1)",
		"busy_timeout(" + strconv.FormatInt(busyTimeout.Milliseconds(), 10) + ")",
	} {
		query.Add("_pragma", pragma)
	}
	uri.RawQuery = query.Encode()
	db, err := sql.Open("sqlite", uri.String())
	if err != nil {
		return nil, err
	}
	db.SetMaxOpenConns(1)
	db.SetMaxIdleConns(1)
	return db, nil
}
