package sqlite

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestMigrationEntrypointCreatesUpOnlyAndRejectsDown(t *testing.T) {
	dir := t.TempDir()
	script := filepath.Join(configDir(t), "entrypoint.sh")
	file := filepath.Join(dir, "20261014140000_probe.sql")
	runner := filepath.Join(dir, "fake-goose.sh")
	source := "#!/bin/sh\nprintf '%s\\n' '-- +goose Up' 'SELECT 1;' '-- +goose Down' 'DROP TABLE probe;' > \"$TEST_MIGRATION_FILE\"\necho \"Created new file: $TEST_MIGRATION_FILE\"\n"
	require.NoError(t, os.WriteFile(runner, []byte(source), 0755))
	cmd := exec.Command("/bin/sh", script)
	cmd.Env = append(os.Environ(), "GOOSE_COMMAND=create", "GOOSE_ENTRYPOINT="+runner, "GOOSE_MIGRATION_DIR="+dir, "TEST_MIGRATION_FILE="+file)
	output, err := cmd.CombinedOutput()
	require.NoError(t, err, string(output))
	data, err := os.ReadFile(file)
	require.NoError(t, err)
	require.Contains(t, string(data), "-- +goose Up")
	require.NotContains(t, string(data), "Down")
	require.NotContains(t, string(data), "DROP TABLE")
	cmd = exec.Command("/bin/sh", script)
	cmd.Env = append(os.Environ(), "GOOSE_COMMAND=down")
	output, err = cmd.CombinedOutput()
	require.Error(t, err)
	require.True(t, strings.Contains(string(output), "Only create, up"))
}
