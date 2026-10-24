//go:build linux || darwin

package filelock

import (
	"errors"
	"os"

	"golang.org/x/sys/unix"

	"github.com/AridanWarlock/pinnAutomizer/pkg/errs"
)

func lock(file *os.File) error {
	err := unix.Flock(int(file.Fd()), unix.LOCK_EX|unix.LOCK_NB)
	if errors.Is(err, unix.EWOULDBLOCK) || errors.Is(err, unix.EAGAIN) {
		return errs.ErrConflict
	}
	return err
}
