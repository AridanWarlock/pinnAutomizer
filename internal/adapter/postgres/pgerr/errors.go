package pgerr

import (
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
)

func IsNotFound(err error) bool {
	return errors.Is(err, pgx.ErrNoRows)
}

func ScanErr(err error) error {
	return fmt.Errorf("scan: %w", err)
}

func ExecErr(err error) error {
	return fmt.Errorf("exec query: %w", err)
}
