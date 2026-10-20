package domain_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/AridanWarlock/pinnAutomizer/pkg/core"
	"github.com/AridanWarlock/pinnAutomizer/pkg/errs"
	"github.com/AridanWarlock/pinnAutomizer/solver/internal/domain/fixtures"
)

func TestDeadlineAndTimeoutTransition(t *testing.T) {
	t.Parallel()
	accepted := fixtures.NewAttempt(t)
	require.False(t, accepted.DeadlineExceeded(time.Now()))
	a := fixtures.NewAttempt(t, fixtures.WithContainer("container"), fixtures.WithStartRequested())
	original := a
	require.False(t, a.DeadlineExceeded(a.Deadline.Add(-time.Nanosecond)))
	require.True(t, a.DeadlineExceeded(*a.Deadline))
	_, err := a.FinishTimedOut(137, a.Deadline.Add(-time.Nanosecond))
	require.ErrorIs(t, err, errs.ErrConflict)
	require.Equal(t, original, a)
	changed, err := a.FinishTimedOut(137, *a.Deadline)
	require.NoError(t, err)
	require.Equal(t, core.Changed(true), changed)
	require.True(t, a.IsFinished())
	require.Equal(t, "execution deadline exceeded", *a.Error)
	require.Equal(t, 137, *a.ExitCode)
	result := a
	changed, err = a.FinishTimedOut(0, time.Now())
	require.NoError(t, err)
	require.Equal(t, core.Changed(false), changed)
	require.Equal(t, result, a)
}
