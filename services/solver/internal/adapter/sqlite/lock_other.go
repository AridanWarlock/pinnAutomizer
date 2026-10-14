//go:build !linux && !darwin

package sqlite

import (
	"fmt"
	"os"
)

func lock(*os.File) error { return fmt.Errorf("solver journal ownership is supported on Linux/macOS") }
