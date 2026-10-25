package mlrunner

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/rs/zerolog"
	"github.com/stretchr/testify/require"

	"github.com/AridanWarlock/pinnAutomizer/pkg/docker"
	"github.com/AridanWarlock/pinnAutomizer/solver/internal/domain/fixtures"
)

func TestAttemptDockerCreateReplayAndExitedContainerIntegration(t *testing.T) {
	if os.Getenv("PINN_TEST_DOCKER") != "1" {
		t.Skip("set PINN_TEST_DOCKER=1 to run a disposable CPU container")
	}
	ctx, cancel := context.WithTimeout(t.Context(), 30*time.Second)
	defer cancel()
	a := fixtures.NewAttempt(t)
	root := t.TempDir()
	data := filepath.Join(root, "data")
	output := filepath.Join(root, "output")
	require.NoError(t, os.MkdirAll(filepath.Join(data, a.TaskID.String()), 0700))
	require.NoError(t, os.MkdirAll(filepath.Join(output, a.TaskID.String()), 0700))
	image := os.Getenv("PINN_TEST_DOCKER_IMAGE")
	if image == "" {
		image = "golang:1.27.3-alpine"
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
		err := runner.docker.(*docker.Client).Remove(cleanup, runner.containerName(a.RunID))
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
	exit, err := runner.docker.Wait(ctx, id)
	require.NoError(t, err)
	require.Zero(t, exit.Code)
	before, err := runner.inspectAttempt(ctx, id, a)
	require.NoError(t, err)
	require.Equal(t, docker.StateExited, before.State)
	require.NoError(t, runner.StartContainer(ctx, a))
	observed, err := runner.ObserveAttempt(ctx, a)
	require.NoError(t, err)
	require.True(t, observed.IsFinished())
	require.Zero(t, *observed.ExitCode)
	require.Nil(t, observed.Error)
	after, err := runner.inspectAttempt(ctx, id, a)
	require.NoError(t, err)
	require.Equal(t, before.StartedAt, after.StartedAt)
	contents, err := os.ReadFile(filepath.Join(output, a.TaskID.String(), "executions"))
	require.NoError(t, err)
	require.Equal(t, "executed\n", string(contents))
}
