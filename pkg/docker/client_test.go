package docker

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/containerd/errdefs"
	"github.com/moby/moby/api/types/container"
	"github.com/moby/moby/client"
	"github.com/stretchr/testify/require"

	"github.com/AridanWarlock/pinnAutomizer/pkg/errs"
)

type transport func(*http.Request) (*http.Response, error)

func (f transport) RoundTrip(r *http.Request) (*http.Response, error) {
	return f(r)
}

func response(t *testing.T, status int, body any) *http.Response {
	t.Helper()
	data, err := json.Marshal(body)
	require.NoError(t, err)
	return &http.Response{
		StatusCode: status,
		Header:     http.Header{"Content-Type": {"application/json"}},
		Body:       io.NopCloser(strings.NewReader(string(data))),
	}
}

func newClient(t *testing.T, send transport) *Client {
	t.Helper()
	cli, err := client.New(client.WithHost("http://docker.test"), client.WithVersion("1.52"),
		client.WithHTTPClient(&http.Client{Transport: send}))
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, cli.Close()) })
	return &Client{cli: cli}
}

func TestCreateMapsConfiguration(t *testing.T) {
	t.Parallel()
	c := newClient(t, func(req *http.Request) (*http.Response, error) {
		require.Equal(t, "/v1.52/containers/create", req.URL.Path)
		require.Equal(t, "attempt", req.URL.Query().Get("name"))
		var body struct {
			container.Config
			HostConfig container.HostConfig
		}
		require.NoError(t, json.NewDecoder(req.Body).Decode(&body))
		require.Equal(t, []string{"python", "run.py"}, body.Cmd)
		require.Equal(t, "image", body.Image)
		require.Equal(t, "run", body.Labels["run_id"])
		require.Equal(t, []string{"A=B"}, body.Env)
		require.False(t, body.HostConfig.AutoRemove)
		require.True(t, body.HostConfig.ReadonlyRootfs)
		require.Equal(t, container.NetworkMode("none"), body.HostConfig.NetworkMode)
		require.Equal(t, []string{"ALL"}, []string(body.HostConfig.CapDrop))
		require.Equal(t, []string{"no-new-privileges:true"}, body.HostConfig.SecurityOpt)
		require.True(t, body.HostConfig.Mounts[0].ReadOnly)
		require.Equal(t, "/input", body.HostConfig.Mounts[0].Source)
		require.EqualValues(t, 1024, body.HostConfig.Mounts[1].TmpfsOptions.SizeBytes)
		require.Equal(t, "nvidia", body.HostConfig.DeviceRequests[0].Driver)
		return response(t, 201, map[string]string{"Id": "created"}), nil
	})
	id, err := c.Create(t.Context(), Config{
		Name:    "attempt",
		Image:   "image",
		Command: []string{"python", "run.py"},
		Env:     []string{"A=B"},
		Labels:  map[string]string{"run_id": "run"},
		Mounts: []Mount{
			{Source: "/input", Target: "/data", ReadOnly: true},
			{Target: "/tmp", TmpfsBytes: 1024, TmpfsMode: 0777},
		},
		GPU:            true,
		ReadonlyRootfs: true,
		NetworkMode:    "none",
		SecurityOpt:    []string{"no-new-privileges:true"},
		CapDrop:        []string{"ALL"},
	})
	require.NoError(t, err)
	require.Equal(t, "created", id)
}

func TestInspectMapsSnapshotAndState(t *testing.T) {
	t.Parallel()
	raw := container.InspectResponse{
		ID:         "id",
		Name:       "/attempt",
		Config:     &container.Config{Image: "image", Labels: map[string]string{"run_id": "run"}},
		HostConfig: &container.HostConfig{},
		State:      &container.State{Status: container.StateExited, StartedAt: "2026-10-20T10:00:00Z", FinishedAt: "2026-10-20T10:01:00Z", ExitCode: 7, Error: "execution failed"},
	}
	c := newClient(t, func(*http.Request) (*http.Response, error) { return response(t, 200, raw), nil })
	snapshot, err := c.Inspect(t.Context(), "id")
	require.NoError(t, err)
	require.True(t, snapshot.HasStarted())
	require.True(t, snapshot.IsExited())
	require.Equal(t, time.Date(2026, 10, 20, 10, 1, 0, 0, time.UTC), snapshot.FinishedAt)
	require.Equal(t, 7, snapshot.ExitCode)
	require.Equal(t, "execution failed", snapshot.Error)
	require.False(t, snapshot.IsCreated())
	require.Equal(t, "run", snapshot.Labels["run_id"])
	require.Equal(t, "image", snapshot.Image)
	require.Equal(t, "/attempt", snapshot.Name)
	raw.State.Status = container.StateCreated
	raw.State.StartedAt = time.Time{}.UTC().Format(time.RFC3339Nano)
	snapshot, err = c.Inspect(t.Context(), "id")
	require.NoError(t, err)
	require.True(t, snapshot.IsCreated())
	require.False(t, snapshot.HasStarted())
	raw.State.FinishedAt = "invalid"
	_, err = c.Inspect(t.Context(), "id")
	require.ErrorContains(t, err, "parse container finish time")
	raw.Config = nil
	_, err = c.Inspect(t.Context(), "id")
	require.ErrorContains(t, err, "incomplete")
}

func TestInspectTranslatesOnlyKnownErrors(t *testing.T) {
	t.Parallel()
	for _, status := range []int{404, 409, 503} {
		c := newClient(t, func(*http.Request) (*http.Response, error) {
			return response(t, status, map[string]string{"message": "failure"}), nil
		})
		_, err := c.Inspect(t.Context(), "id")
		require.Error(t, err)
		if status == 404 {
			require.ErrorIs(t, err, errs.ErrNotFound)
		}
		if status == 409 {
			require.ErrorIs(t, err, errs.ErrConflict)
		}
		if status == 503 {
			require.ErrorContains(t, err, "unexpected docker error:")
			require.ErrorContains(t, err, "failure")
			require.True(t, errdefs.IsUnavailable(err), "preserve the original Docker error category")
		}
	}
}

func TestWaitMapsExitAndCancellation(t *testing.T) {
	t.Parallel()
	c := newClient(t, func(req *http.Request) (*http.Response, error) {
		if err := req.Context().Err(); err != nil {
			return nil, err
		}
		return response(t, 200, container.WaitResponse{StatusCode: 7, Error: &container.WaitExitError{Message: "failed"}}), nil
	})
	exit, err := c.Wait(t.Context(), "id")
	require.NoError(t, err)
	require.Equal(t, Exit{Code: 7, Error: "failed"}, exit)
	ctx, cancel := context.WithCancel(t.Context())
	cancel()
	_, err = c.Wait(ctx, "id")
	require.ErrorIs(t, err, context.Canceled)
}

func TestDockerErrorPreservesSuccess(t *testing.T) {
	t.Parallel()
	require.NoError(t, dockerError(nil))
}

func TestDockerErrorPreservesContext(t *testing.T) {
	t.Parallel()
	for _, original := range []error{context.Canceled, context.DeadlineExceeded} {
		require.Equal(t, original, dockerError(original))
	}
}

func TestDockerErrorWrapsUnknownFailure(t *testing.T) {
	t.Parallel()
	original := errors.New("connection lost")
	err := dockerError(original)
	require.ErrorIs(t, err, original)
	require.EqualError(t, err, "unexpected docker error: connection lost")
}

func TestStopMapsBoundedGraceAndPreservesErrors(t *testing.T) {
	t.Parallel()
	c := newClient(t, func(req *http.Request) (*http.Response, error) {
		require.Equal(t, "/v1.52/containers/id/stop", req.URL.Path)
		require.Equal(t, "2", req.URL.Query().Get("t"))
		return response(t, http.StatusNotFound, map[string]string{"message": "missing"}), nil
	})
	require.ErrorIs(t, c.Stop(t.Context(), "id", 1500*time.Millisecond), errs.ErrNotFound)
	require.ErrorIs(t, c.Stop(t.Context(), "id", -time.Second), errs.ErrInvalidArgument)
}
