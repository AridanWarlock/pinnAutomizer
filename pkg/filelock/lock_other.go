//go:build !linux && !darwin

package filelock

import (
	"fmt"
	"os"
)

func lock(*os.File) error {
	return fmt.Errorf("File ownership is supported on Linux/macOS")
}
