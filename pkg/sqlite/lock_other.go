//go:build !linux && !darwin

package sqlite

import (
	"fmt"
	"os"
)

func lock(*os.File) error { return fmt.Errorf("SQLite ownership is supported on Linux/macOS") }
