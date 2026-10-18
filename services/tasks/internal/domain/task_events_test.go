package domain

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestTaskKeepsFirstResultAcrossEventOrders(t *testing.T) {
	failure := "solver failed"
	for _, tc := range []struct {
		name   string
		events []string
		status TaskStatus
		err    *string
	}{
		{"start then success", []string{"start", "success", "failure", "start"}, TaskStatusDone, nil},
		{"start then failure", []string{"start", "failure", "success", "start"}, TaskStatusError, &failure},
		{"success before start", []string{"success", "start", "failure"}, TaskStatusDone, nil},
		{"failure before start", []string{"failure", "start", "success"}, TaskStatusError, &failure},
		{"duplicate success", []string{"start", "start", "success", "success"}, TaskStatusDone, nil},
		{"duplicate failure", []string{"failure", "failure", "start", "start"}, TaskStatusError, &failure},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			task := Task{Status: TaskStatusInQueue}
			for _, event := range tc.events {
				var err error
				switch event {
				case "start":
					_, err = task.MarkRunning()
				case "success":
					_, err = task.Complete(nil)
				case "failure":
					_, err = task.Complete(&failure)
				default:
					t.Fatalf("unknown test event: %s", event)
				}
				require.NoError(t, err, "event %s", event)
			}
			require.Equal(t, tc.status, task.Status)
			require.Equal(t, tc.err, task.Error)
		})
	}
}
