package domain

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestTaskMarkRunning(t *testing.T) {
	for _, tc := range []struct {
		status  TaskStatus
		changed bool
		want    TaskStatus
		err     error
	}{
		{TaskStatusCreated, false, TaskStatusCreated, ErrInvalidTaskTransition},
		{TaskStatusInQueue, true, TaskStatusRunning, nil},
		{TaskStatusRunning, false, TaskStatusRunning, nil},
		{TaskStatusDone, false, TaskStatusDone, nil},
		{TaskStatusError, false, TaskStatusError, nil},
	} {
		t.Run(string(tc.status), func(t *testing.T) {
			task := Task{Status: tc.status}
			changed, err := task.MarkRunning()
			assert.Equal(t, tc.changed, changed)
			assert.ErrorIs(t, err, tc.err)
			assert.Equal(t, tc.want, task.Status)
		})
	}
}

func TestTaskComplete(t *testing.T) {
	failure := "solver failed"
	previous := "original error"
	for _, status := range []TaskStatus{TaskStatusCreated, TaskStatusInQueue, TaskStatusRunning, TaskStatusDone, TaskStatusError} {
		for _, runError := range []*string{nil, &failure} {
			name := string(status) + "/success"
			if runError != nil {
				name = string(status) + "/failure"
			}
			t.Run(name, func(t *testing.T) {
				task := Task{Status: status, Error: &previous}
				changed, err := task.Complete(runError)
				switch status {
				case TaskStatusCreated:
					assert.ErrorIs(t, err, ErrInvalidTaskTransition)
					assert.False(t, changed)
					assert.Equal(t, status, task.Status)
					assert.Equal(t, &previous, task.Error)
				case TaskStatusDone, TaskStatusError:
					assert.NoError(t, err)
					assert.False(t, changed)
					assert.Equal(t, status, task.Status)
					assert.Equal(t, &previous, task.Error)
				default:
					assert.NoError(t, err)
					assert.True(t, changed)
					assert.Equal(t, runError, task.Error)
					want := TaskStatusDone
					if runError != nil {
						want = TaskStatusError
					}
					assert.Equal(t, want, task.Status)
				}
			})
		}
	}
}

func TestTaskCanDelete(t *testing.T) {
	for _, status := range []TaskStatus{TaskStatusCreated, TaskStatusInQueue, TaskStatusRunning, TaskStatusDone, TaskStatusError, "unknown"} {
		t.Run(string(status), func(t *testing.T) {
			task := Task{Status: status}
			assert.Equal(t, status == TaskStatusCreated || status == TaskStatusDone || status == TaskStatusError, task.CanDelete())
		})
	}
}
