package domain_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/AridanWarlock/pinnAutomizer/pkg/core"
	"github.com/AridanWarlock/pinnAutomizer/pkg/errs"
	"github.com/AridanWarlock/pinnAutomizer/solver/internal/domain"
	"github.com/AridanWarlock/pinnAutomizer/solver/internal/domain/fixtures"
)

func TestCleanupRequiresFinishedContainerAndPreservesResult(t *testing.T) {
	t.Parallel()
	a := fixtures.NewAttempt(t, fixtures.WithContainer("id"))
	original := a
	changed, err := a.MarkCleaned(time.Now())
	require.ErrorIs(t, err, errs.ErrConflict)
	require.Equal(t, core.Changed(false), changed)
	require.Equal(t, original, a)

	code := 0
	finished := time.Now()
	_, err = a.Finish(&code, nil, finished)
	require.NoError(t, err)
	require.True(t, a.NeedsCleanup())
	original = a
	_, err = a.MarkCleaned(finished.Add(-time.Second))
	require.ErrorIs(t, err, errs.ErrInvalidArgument)
	require.Equal(t, original, a, "failed transition must leave the entity unchanged")

	changed, err = a.MarkCleaned(finished.Add(time.Second))
	require.NoError(t, err)
	require.Equal(t, core.Changed(true), changed)
	require.False(t, a.NeedsCleanup())
	require.Equal(t, original.CompletedAt, a.CompletedAt)
	require.Equal(t, original.ExitCode, a.ExitCode)
	cleaned := a
	changed, err = a.MarkCleaned(finished.Add(time.Hour))
	require.NoError(t, err)
	require.Equal(t, core.Changed(false), changed)
	require.Equal(t, cleaned, a, "replay must preserve the first cleanup timestamp")

	snapshot := domain.AttemptSnapshot(a)
	snapshot.State = domain.AttemptRunning
	_, err = domain.RestoreAttempt(snapshot)
	require.Error(t, err, "unfinished attempt cannot have a cleanup marker")
}

func TestFinishedAttemptWithoutContainerNeedsNoCleanup(t *testing.T) {
	t.Parallel()
	a := fixtures.NewAttempt(t)
	failure := "input preparation failed"
	_, err := a.Finish(nil, &failure, time.Now())
	require.NoError(t, err)
	require.False(t, a.NeedsCleanup())
	_, err = a.MarkCleaned(time.Now())
	require.ErrorIs(t, err, errs.ErrConflict)
}
