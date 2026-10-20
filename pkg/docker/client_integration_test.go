package docker

import (
	"context"
	"io"
	"os"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

func TestDockerClientLifecycleIntegration(t *testing.T) {
	if os.Getenv("PINN_TEST_DOCKER") != "1" {
		t.Skip("set PINN_TEST_DOCKER=1 to run a disposable CPU container")
	}
	ctx, cancel := context.WithTimeout(t.Context(), 30*time.Second)
	defer cancel()
	c, err := New()
	require.NoError(t, err)
	defer c.Close()
	name := "pinn-client-test-" + uuid.NewString()
	defer func() {
		cleanup, stop := context.WithTimeout(context.Background(), 10*time.Second)
		defer stop()
		require.NoError(t, c.Remove(cleanup, name))
	}()
	image := os.Getenv("PINN_TEST_DOCKER_IMAGE")
	if image == "" {
		image = "golang:1.26.1-alpine"
	}
	id, err := c.Create(ctx, Config{
		Name:           name,
		Image:          image,
		Command:        []string{"sh", "-c", "echo shared-docker-ok"},
		Labels:         map[string]string{"pinn.test": name},
		ReadonlyRootfs: true,
		NetworkMode:    "none",
	})
	require.NoError(t, err)
	snapshot, err := c.Inspect(ctx, id)
	require.NoError(t, err)
	require.True(t, snapshot.IsCreated())
	require.False(t, snapshot.HasStarted())
	require.False(t, snapshot.AutoRemove)
	require.Equal(t, name, snapshot.Labels["pinn.test"])
	require.NoError(t, c.Start(ctx, id))
	exit, err := c.Wait(ctx, id)
	require.NoError(t, err)
	require.Zero(t, exit.Code)
	snapshot, err = c.Inspect(ctx, id)
	require.NoError(t, err)
	require.True(t, snapshot.HasStarted())
	logs, err := c.Logs(ctx, id)
	require.NoError(t, err)
	defer logs.Close()
	contents, err := io.ReadAll(io.LimitReader(logs, 64*1024))
	require.NoError(t, err)
	require.Contains(t, string(contents), "shared-docker-ok")
}
