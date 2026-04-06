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
	return fmt.Errorf("scan error: %w", err)
}
