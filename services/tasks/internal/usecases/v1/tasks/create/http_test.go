package tasksCreate

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/AridanWarlock/pinnAutomizer/pkg/core/corefixtures"
	"github.com/AridanWarlock/pinnAutomizer/pkg/test"
	"github.com/AridanWarlock/pinnAutomizer/tasks/internal/domain"
)

func TestHttpHandler_CreateTask(t *testing.T) {
	for _, mode := range []domain.TaskMode{domain.TaskModeTrain, domain.TaskModeRetrain, domain.TaskModePredict} {
		t.Run(string(mode), func(t *testing.T) {
			var body bytes.Buffer
			form := multipart.NewWriter(&body)
			require.NoError(t, form.WriteField("data", fmt.Sprintf(`{"name":"task","mode":%q}`, mode)))
			for _, name := range mode.RequiredFiles() {
				file, err := form.CreateFormFile(name, name)
				require.NoError(t, err)
				_, err = io.WriteString(file, "content:"+name)
				require.NoError(t, err)
			}
			require.NoError(t, form.Close())
			var opened []domain.TaskFile
			usecase := &MockUsecase{CreateTaskFunc: func(ctx context.Context, in Input) (Output, error) {
				assert.Equal(t, mode, in.Mode)
				assert.Len(t, in.Files, len(mode.RequiredFiles()))
				seen := make(map[string]bool)
				for _, file := range in.Files {
					assert.False(t, seen[file.Name], "duplicate file")
					seen[file.Name] = true
					data, err := io.ReadAll(file.File)
					require.NoError(t, err)
					assert.Equal(t, "content:"+file.Name, string(data))
				}
				opened = in.Files
				task, err := domain.NewTask(in.Name, corefixtures.NewAuthInfo().UserID, mode, nil)
				return Output{Task: task}, err
			}}
			req := httptest.NewRequest(http.MethodPost, "/tasks", &body).WithContext(test.ContextWithZeroLogger())
			req.Header.Set("Content-Type", form.FormDataContentType())
			res := httptest.NewRecorder()
			NewHttpHandler(usecase).CreateTask(res, req)
			assert.Equal(t, http.StatusCreated, res.Code)
			assert.Len(t, usecase.CreateTaskCalls(), 1)
			assert.NotNil(t, opened)
		})
	}
}

type zeroReader struct{}

func (zeroReader) Read(p []byte) (int, error) { clear(p); return len(p), nil }

func TestHttpHandler_CreateTaskRejectsOversizedBody(t *testing.T) {
	// Stream an oversized part instead of allocating a 200 MiB byte slice.
	tempDir := t.TempDir()
	t.Setenv("TMPDIR", tempDir)
	const boundary = "upload-limit-test"
	prefix := "--" + boundary + "\r\nContent-Disposition: form-data; name=\"data\"\r\n\r\n{\"name\":\"task\",\"mode\":\"train\"}\r\n--" + boundary + "\r\nContent-Disposition: form-data; name=\"data.mat\"; filename=\"data.mat\"\r\nContent-Type: application/octet-stream\r\n\r\n"
	body := io.MultiReader(strings.NewReader(prefix), io.LimitReader(zeroReader{}, maxUploadSize+1), strings.NewReader("\r\n--"+boundary+"--\r\n"))
	req := httptest.NewRequest(http.MethodPost, "/tasks", body).WithContext(test.ContextWithZeroLogger())
	req.Header.Set("Content-Type", "multipart/form-data; boundary="+boundary)
	usecase := &MockUsecase{}
	res := httptest.NewRecorder()
	NewHttpHandler(usecase).CreateTask(res, req)
	assert.Equal(t, http.StatusBadRequest, res.Code)
	assert.Contains(t, res.Body.String(), "request body too large")
	assert.Empty(t, usecase.CreateTaskCalls())
	entries, err := os.ReadDir(tempDir)
	require.NoError(t, err)
	assert.Empty(t, entries, "temporary multipart files must be removed")
}

type trackedFile struct {
	io.Reader
	closed bool
}

func (f *trackedFile) Close() error { f.closed = true; return nil }

func TestCollectTaskFilesClosesPartialCollection(t *testing.T) {
	first := &trackedFile{Reader: strings.NewReader("data")}
	files, err := collectTaskFiles(domain.TaskModeTrain, func(name string) (io.ReadCloser, error) {
		if name == "data.mat" {
			return first, nil
		}
		return nil, http.ErrMissingFile
	})
	require.Error(t, err)
	assert.Nil(t, files)
	assert.True(t, first.closed)
}
