package filestore

import (
	"bytes"
	"errors"
	"io"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/AridanWarlock/pinnAutomizer/tasks/internal/domain"
)

type testDestination struct {
	bytes.Buffer
	closeError error
	closed     bool
}

func (w *testDestination) Close() error {
	w.closed = true
	return w.closeError
}

type failedSource struct {
	err error
}

func (r failedSource) Read([]byte) (int, error) {
	return 0, r.err
}

func TestStoreUsesFileServiceAndPropagatesWriteAndCloseFailures(t *testing.T) {
	t.Parallel()
	for _, stage := range []string{"success", "mkdir", "create", "copy", "close"} {
		t.Run(stage, func(t *testing.T) {
			t.Parallel()
			failure := errors.New("injected failure")
			destination := &testDestination{}
			var reader io.Reader = strings.NewReader("data")
			if stage == "copy" {
				reader = failedSource{err: failure}
			}
			if stage == "close" {
				destination.closeError = failure
			}
			service := &MockFileService{
				MkdirAllFunc: func(path string) error {
					require.Equal(t, "/tasks/id", path)
					if stage == "mkdir" {
						return failure
					}
					return nil
				},
				CreateFunc: func(path string) (io.WriteCloser, error) {
					require.Equal(t, "/tasks/id/input", path)
					if stage == "create" {
						return nil, failure
					}
					return destination, nil
				},
			}
			err := NewFileStore(service).Store("/tasks/id", []domain.TaskFile{
				{Name: "input", File: io.NopCloser(reader)},
			})
			if stage == "success" {
				require.NoError(t, err)
				require.Equal(t, "data", destination.String())
			} else {
				require.ErrorIs(t, err, failure)
			}
			require.Equal(t, stage != "mkdir" && stage != "create", destination.closed)
		})
	}
}
