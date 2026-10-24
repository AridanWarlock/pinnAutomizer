//go:build linux || darwin

package fileservice

import (
	"context"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/AridanWarlock/pinnAutomizer/pkg/errs"
)

func TestDirectoryLockProtectsOperationsAcrossClients(t *testing.T) {
	path := filepath.Join(t.TempDir(), "directory")
	first, second := NewFileService(), NewFileService()
	require.NoError(t, first.WithDirectoryLock(t.Context(), path, func(ctx context.Context) error {
		returnErr := second.WithDirectoryLock(ctx, path, func(context.Context) error {
			t.Fatal("another client entered the locked directory")
			return nil
		})
		require.ErrorIs(t, returnErr, errs.ErrConflict)
		return nil
	}))
	require.NoError(t, second.WithDirectoryLock(t.Context(), path, func(context.Context) error { return nil }))
	require.NoDirExists(t, path, "acquiring ownership must not create the task directory")
}
