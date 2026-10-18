package mlrunner

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/moby/moby/api/types/container"
	"github.com/moby/moby/client"
	"github.com/rs/zerolog"
	"github.com/stretchr/testify/require"
)

func TestAttemptDockerCreateReplayAndExitedContainerIntegration(t *testing.T) {
	if os.Getenv("PINN_TEST_DOCKER") != "1" {
		t.Skip("set PINN_TEST_DOCKER=1 to run a disposable CPU container")
	}
	ctx, cancel := context.WithTimeout(t.Context(), 30*time.Second)
	defer cancel()
	a := testAttempt(t)
	root := t.TempDir()
	data := filepath.Join(root, "data")
	output := filepath.Join(root, "output")
	require.NoError(t, os.MkdirAll(filepath.Join(data, a.TaskID.String()), 0700))
	require.NoError(t, os.MkdirAll(filepath.Join(output, a.TaskID.String()), 0700))
	image := os.Getenv("PINN_TEST_DOCKER_IMAGE")
	if image == "" {
		image = "golang:1.26.1-alpine"
	}
	runner, err := NewPinnRunner(Config{
		Image:              image,
		HostTasksDataDir:   data,
		HostTasksOutputDir: output,
	}, zerolog.Nop())
	require.NoError(t, err)
	defer runner.Close()
	// Cleanup by stable name even if Create committed but its response was lost.
	defer func() {
		cleanup, stop := context.WithTimeout(context.Background(), 10*time.Second)
		defer stop()
		_, err := runner.cli.ContainerRemove(cleanup, runner.attemptName(a), client.ContainerRemoveOptions{Force: true})
		require.NoError(t, err)
	}()
	command := []string{"sh", "-c", "echo executed >> /task_output/executions"}
	id, err := runner.ensureContainer(ctx, a, command)
	require.NoError(t, err)
	repeatedID, err := runner.ensureContainer(ctx, a, command)
	require.NoError(t, err)
	require.Equal(t, id, repeatedID)
	_, err = a.RecordContainer(id)
	require.NoError(t, err)
	_, err = a.RequestStart(time.Now(), time.Minute)
	require.NoError(t, err)
	require.NoError(t, runner.StartContainer(ctx, a))
	wait := runner.cli.ContainerWait(ctx, id, client.ContainerWaitOptions{Condition: container.WaitConditionNotRunning})
	select {
	case result := <-wait.Result:
		require.EqualValues(t, 0, result.StatusCode)
	case err := <-wait.Error:
		require.NoError(t, err)
	case <-ctx.Done():
		t.Fatal(ctx.Err())
	}
	before, err := runner.inspectAttempt(ctx, id, a)
	require.NoError(t, err)
	require.Equal(t, container.StateExited, before.State.Status)
	require.NoError(t, runner.StartContainer(ctx, a))
	after, err := runner.inspectAttempt(ctx, id, a)
	require.NoError(t, err)
	require.Equal(t, before.State.StartedAt, after.State.StartedAt)
	contents, err := os.ReadFile(filepath.Join(output, a.TaskID.String(), "executions"))
	require.NoError(t, err)
	require.Equal(t, "executed\n", string(contents))
}
