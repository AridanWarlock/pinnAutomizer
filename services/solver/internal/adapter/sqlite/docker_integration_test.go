package sqlite

import (
	"bufio"
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

func TestGooseDockerUsesOwnershipLockIntegration(t *testing.T) {
	if os.Getenv("PINN_TEST_DOCKER") != "1" {
		t.Skip("set PINN_TEST_DOCKER=1 to test the disposable migration container")
	}
	ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
	defer cancel()
	dir := t.TempDir()
	migrations := configDir(t)
	image := "kukymbr/goose-docker:3.27.0"
	goose := func() ([]byte, error) {
		return exec.CommandContext(ctx, "docker", "run", "--rm",
			"-v", dir+":/journal",
			"-v", migrations+":/migrations:ro",
			"-v", filepath.Join(migrations, "entrypoint.sh")+":/goose-solver/entrypoint.sh:ro",
			"-e", "GOOSE_DRIVER=sqlite3",
			"-e", "GOOSE_DBSTRING=file:/journal/journal.db",
			"-e", "GOOSE_MIGRATION_DIR=/migrations",
			"-e", "GOOSE_COMMAND=up",
			"--entrypoint", "/bin/sh", image, "/goose-solver/entrypoint.sh").CombinedOutput()
	}
	output, err := goose()
	require.NoError(t, err, "%s", output)
	cfg := Config{Path: filepath.Join(dir, "journal.db"), BusyTimeout: time.Second, MigrationsDir: migrations}
	repo, err := New(cfg)
	require.NoError(t, err)
	require.NoError(t, repo.Close())

	// Both ownership contenders run on Linux; host macOS and Docker VM flock
	// interoperability is not assumed.
	name := "pinn-test-owner-" + uuid.NewString()
	output, err = exec.CommandContext(ctx, "docker", "run", "--rm", "-d", "--name", name,
		"-v", dir+":/journal", "--entrypoint", "flock", image,
		"--exclusive", "/journal/journal.db.lock", "/bin/sh", "-c", "echo ready; sleep 120").CombinedOutput()
	require.NoError(t, err, "%s", output)
	t.Cleanup(func() {
		cleanup, stop := context.WithTimeout(context.Background(), 10*time.Second)
		defer stop()
		_ = exec.CommandContext(cleanup, "docker", "rm", "-f", name).Run()
	})
	logs := exec.CommandContext(ctx, "docker", "logs", "--follow", name)
	stdout, err := logs.StdoutPipe()
	require.NoError(t, err)
	require.NoError(t, logs.Start())
	line, readErr := bufio.NewReader(stdout).ReadString('\n')
	_ = logs.Process.Kill()
	_ = logs.Wait()
	require.NoError(t, readErr)
	require.Equal(t, "ready", strings.TrimSpace(line))
	_, err = goose()
	require.Error(t, err, "migration must fail while another owner holds the lock")
	output, err = exec.CommandContext(ctx, "docker", "rm", "-f", name).CombinedOutput()
	require.NoError(t, err, "%s", output)
	output, err = goose()
	require.NoError(t, err, "%s", output)
	repo, err = New(cfg)
	require.NoError(t, err)
	require.NoError(t, repo.Close())
}
