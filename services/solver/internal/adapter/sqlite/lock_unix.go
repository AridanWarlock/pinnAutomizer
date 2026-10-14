//go:build linux || darwin

package sqlite

import (
	"errors"
	"os"

	"golang.org/x/sys/unix"
)

func lock(file *os.File) error {
	err := unix.Flock(int(file.Fd()), unix.LOCK_EX|unix.LOCK_NB)
	if errors.Is(err, unix.EWOULDBLOCK) || errors.Is(err, unix.EAGAIN) {
		return ErrAlreadyOwned
	}
	return err
}
