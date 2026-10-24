// Package filelock serializes cooperating processes sharing a filesystem.
package filelock

import (
	"errors"
	"os"
)

// WithLock attempts an exclusive advisory lock without waiting. The stable lock
// file must not be removed: unlinking it would allow locking a different inode.
func WithLock(path string, process func() error) (err error) {
	file, err := os.OpenFile(path, os.O_CREATE|os.O_RDWR, 0600)
	if err != nil {
		return err
	}
	defer func() { err = errors.Join(err, file.Close()) }()
	if err := lock(file); err != nil {
		return err
	}
	return process()
}
