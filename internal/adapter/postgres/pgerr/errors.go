package pgerr

import (
	"errors"
	"fmt"

	"github.com/AridanWarlock/pinnAutomizer/internal/errs"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

func IsNotFound(err error) bool {
	return errors.Is(err, pgx.ErrNoRows)
}

func ScanErr(err error) error {
	if err == nil {
		return nil
	}

	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) {
		switch pgErr.Code {
		case "23505", "23503":
			return errs.ErrConflict
		}
	}

	return fmt.Errorf("scan err: %w", err)
}
