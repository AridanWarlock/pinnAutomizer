package domain

import (
	"path/filepath"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

func TestFileCleanupOnlyAcceptsOwnedTaskDirectories(t *testing.T) {
	t.Parallel()
	id := uuid.New()
	data := filepath.Join("/tasks_data", id.String())
	output := filepath.Join("/tasks_output", id.String())
	job, err := NewFileCleanup(id, data, output)
	require.NoError(t, err)
	require.Equal(t, id, job.TaskID)
	require.Equal(t, data, job.DataPath)
	require.Equal(t, output, job.OutputPath)
	for _, invalid := range []struct {
		name   string
		id     uuid.UUID
		data   string
		output string
	}{
		{name: "missing task", data: data, output: output},
		{name: "missing data", id: id, output: output},
		{name: "root directory", id: id, data: "/", output: output},
		{name: "relative path", id: id, data: id.String(), output: output},
		{name: "foreign task", id: id, data: filepath.Join("/tasks_data", uuid.NewString()), output: output},
		{name: "same directories", id: id, data: data, output: data},
	} {
		t.Run(invalid.name, func(t *testing.T) {
			_, err := NewFileCleanup(invalid.id, invalid.data, invalid.output)
			require.Error(t, err, "id=%s data=%q output=%q", invalid.id, invalid.data, invalid.output)
		})
	}
}
