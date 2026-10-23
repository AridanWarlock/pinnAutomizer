//go:build linux || darwin

package filelock

import (
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/AridanWarlock/pinnAutomizer/pkg/errs"
)

func TestExclusiveLockReleasesAfterFailure(t *testing.T) {
	path := filepath.Join(t.TempDir(), "owner.lock")
	failure := errors.New("operation failed")
	err := WithLock(path, func() error {
		require.ErrorIs(t, WithLock(path, func() error {
			t.Fatal("second owner must not enter")
			return nil
		}), errs.ErrConflict)
		return failure
	})
	require.ErrorIs(t, err, failure)
	require.NoError(t, WithLock(path, func() error { return nil }))
	require.FileExists(t, path, "stable inode must survive unlocking")
}

func TestLockReleasesAfterProcessDeath(t *testing.T) {
	if path := os.Getenv("PINN_FILELOCK_CHILD_PATH"); path != "" {
		err := WithLock(path, func() error {
			_, err := os.Stdout.Write([]byte{1})
			if err != nil {
				return err
			}
			// The parent kills this owner; no defer can run.
			<-time.After(time.Hour)
			return nil
		})
		if err != nil {
			os.Exit(2)
		}
		return
	}
	path := filepath.Join(t.TempDir(), "owner.lock")
	cmd := exec.CommandContext(t.Context(), os.Args[0], "-test.run=^TestLockReleasesAfterProcessDeath$")
	cmd.Env = append(os.Environ(), "PINN_FILELOCK_CHILD_PATH="+path)
	ready, err := cmd.StdoutPipe()
	require.NoError(t, err)
	require.NoError(t, cmd.Start())
	defer func() { _ = cmd.Process.Kill(); _ = cmd.Wait() }()
	signal := make([]byte, 1)
	_, err = ready.Read(signal)
	require.NoError(t, err)
	require.ErrorIs(t, WithLock(path, func() error { return nil }), errs.ErrConflict)
	require.NoError(t, cmd.Process.Kill())
	require.Error(t, cmd.Wait())
	require.NoError(t, WithLock(path, func() error { return nil }))
}
