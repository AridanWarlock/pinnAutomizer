package domain

import (
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"

	"github.com/AridanWarlock/pinnAutomizer/pkg/errs"
)

func TestAttemptTransitionsPreserveInvariantAndOriginalDeadline(t *testing.T) {
	now := time.Now().UTC()
	a, err := NewAttempt(uuid.New(), uuid.New(), "train", now)
	require.NoError(t, err)
	_, err = a.RequestStart(now, time.Minute)
	require.ErrorIs(t, err, errs.ErrConflict)
	require.Equal(t, AttemptAccepted, a.State)
	changed, err := a.RecordContainer("container")
	require.NoError(t, err)
	require.True(t, bool(changed))
	require.NoError(t, a.Validate())
	changed, err = a.RequestStart(now, time.Minute)
	require.NoError(t, err)
	require.True(t, bool(changed))
	deadline := *a.Deadline
	changed, err = a.RequestStart(now.Add(time.Minute), time.Hour)
	require.NoError(t, err)
	require.False(t, bool(changed))
	require.Equal(t, deadline, *a.Deadline)
	require.NoError(t, a.Validate())
}

func TestAttemptRejectsInvalidTransitionWithoutMutation(t *testing.T) {
	a, err := NewAttempt(uuid.New(), uuid.New(), "predict", time.Now())
	require.NoError(t, err)
	before := a
	_, err = a.RecordContainer("")
	require.ErrorIs(t, err, errs.ErrInvalidArgument)
	require.Equal(t, before, a)
	_, err = a.Finish(nil, nil, time.Now())
	require.ErrorIs(t, err, errs.ErrInvalidArgument)
	require.Equal(t, before, a)
}

func TestRestoreAttemptRejectsInconsistentSnapshot(t *testing.T) {
	a, err := NewAttempt(uuid.New(), uuid.New(), "train", time.Now())
	require.NoError(t, err)
	snapshot := AttemptSnapshot(a)
	snapshot.State = AttemptStartRequested
	_, err = RestoreAttempt(snapshot)
	require.ErrorIs(t, err, errs.ErrInvalidArgument)
	snapshot.ContainerID = "container"
	_, err = RestoreAttempt(snapshot)
	require.ErrorIs(t, err, errs.ErrInvalidArgument)
	deadline := time.Now().Add(time.Hour)
	snapshot.Deadline = &deadline
	restored, err := RestoreAttempt(snapshot)
	require.NoError(t, err)
	require.NoError(t, restored.Validate())
}

func TestAttemptFinishKeepsFirstResultAndCopiesInput(t *testing.T) {
	a, err := NewAttempt(uuid.New(), uuid.New(), "train", time.Now())
	require.NoError(t, err)
	code := 0
	_, err = a.Finish(&code, nil, time.Now())
	require.NoError(t, err)
	code = 42
	require.Equal(t, 0, *a.ExitCode)
	failure := "late failure"
	changed, err := a.Finish(nil, &failure, time.Now())
	require.NoError(t, err)
	require.False(t, bool(changed))
	require.Nil(t, a.Error)
	require.NoError(t, a.Validate())
}
