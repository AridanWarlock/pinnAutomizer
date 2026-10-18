package mlrunner

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/moby/moby/api/types/container"
	"github.com/moby/moby/client"
	"github.com/stretchr/testify/require"

	"github.com/AridanWarlock/pinnAutomizer/pkg/errs"
	"github.com/AridanWarlock/pinnAutomizer/solver/internal/domain"
)

type dockerTransport func(*http.Request) (*http.Response, error)

func (f dockerTransport) RoundTrip(r *http.Request) (*http.Response, error) {
	return f(r)
}

func dockerResponse(t *testing.T, status int, body any) *http.Response {
	t.Helper()
	data, err := json.Marshal(body)
	require.NoError(t, err)
	return &http.Response{
		StatusCode: status,
		Header:     http.Header{"Content-Type": {"application/json"}},
		Body:       io.NopCloser(strings.NewReader(string(data))),
	}
}

func testAttempt(t *testing.T) domain.Attempt {
	t.Helper()
	a, err := domain.NewAttempt(uuid.New(), uuid.New(), domain.MlTaskModePredict, time.Now())
	require.NoError(t, err)
	return a
}

func testRunner(t *testing.T, transport dockerTransport) *PinnRunner {
	t.Helper()
	cli, err := client.New(client.WithHost("http://docker.test"), client.WithVersion("1.52"),
		client.WithHTTPClient(&http.Client{Transport: transport}))
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, cli.Close()) })
	return &PinnRunner{cli: cli, pinnImage: "test-image"}
}

func ownedContainer(a domain.Attempt) container.InspectResponse {
	return container.InspectResponse{
		ID:   "container-id",
		Name: "/" + (Config{}).ContainerName(a.RunID),
		Config: &container.Config{
			Image: "test-image",
			Labels: map[string]string{
				runLabel:  a.RunID.String(),
				taskLabel: a.TaskID.String(),
				modeLabel: a.Mode,
			},
		},
		HostConfig: &container.HostConfig{},
		State:      &container.State{Status: container.StateCreated},
	}
}

func TestEnsureContainerReconcilesLostCreateResponse(t *testing.T) {
	t.Parallel()
	a := testAttempt(t)
	existing := ownedContainer(a)
	created := false
	creates := 0
	runner := testRunner(t, func(req *http.Request) (*http.Response, error) {
		if req.Method == http.MethodGet {
			if !created {
				return dockerResponse(t, http.StatusNotFound, map[string]string{"message": "missing"}), nil
			}
			return dockerResponse(t, http.StatusOK, existing), nil
		}
		require.Equal(t, "/v1.52/containers/create", req.URL.Path)
		require.Equal(t, (Config{}).ContainerName(a.RunID), req.URL.Query().Get("name"))
		var body struct {
			container.Config
			HostConfig container.HostConfig
		}
		require.NoError(t, json.NewDecoder(req.Body).Decode(&body))
		require.False(t, body.HostConfig.AutoRemove)
		require.Equal(t, a.RunID.String(), body.Labels[runLabel])
		require.Equal(t, []string{"python", "run.py", "--mod=predict"}, body.Cmd)
		creates++
		created = true
		return dockerResponse(t, http.StatusServiceUnavailable, map[string]string{"message": "response lost"}), nil
	})
	for range 2 {
		id, err := runner.EnsureContainer(t.Context(), a)
		require.NoError(t, err)
		require.Equal(t, existing.ID, id)
	}
	require.Equal(t, 1, creates)
}

func TestEnsureContainerDoesNotCreateOnUnavailableInspect(t *testing.T) {
	t.Parallel()
	runner := testRunner(t, func(req *http.Request) (*http.Response, error) {
		require.Equal(t, http.MethodGet, req.Method)
		return dockerResponse(t, http.StatusServiceUnavailable, map[string]string{"message": "unavailable"}), nil
	})
	_, err := runner.EnsureContainer(t.Context(), testAttempt(t))
	require.Error(t, err)
}

func TestEnsureContainerRejectsForeignOrUnsafeContainer(t *testing.T) {
	t.Parallel()
	for _, scenario := range []struct {
		name   string
		change func(*container.InspectResponse)
	}{
		{"foreign run", func(c *container.InspectResponse) { c.Config.Labels[runLabel] = uuid.NewString() }},
		{"different mode", func(c *container.InspectResponse) { c.Config.Labels[modeLabel] = "train" }},
		{"different image", func(c *container.InspectResponse) { c.Config.Image = "other" }},
		{"automatic removal", func(c *container.InspectResponse) { c.HostConfig.AutoRemove = true }},
		{"automatic restart", func(c *container.InspectResponse) { c.HostConfig.RestartPolicy.Name = container.RestartPolicyAlways }},
	} {
		t.Run(scenario.name, func(t *testing.T) {
			a := testAttempt(t)
			existing := ownedContainer(a)
			scenario.change(&existing)
			runner := testRunner(t, func(req *http.Request) (*http.Response, error) {
				require.Equal(t, http.MethodGet, req.Method)
				return dockerResponse(t, http.StatusOK, existing), nil
			})
			_, err := runner.EnsureContainer(t.Context(), a)
			require.ErrorIs(t, err, errs.ErrConflict)
		})
	}
}

func TestStartContainerNeverRetriesStart(t *testing.T) {
	t.Parallel()
	for _, state := range []container.ContainerState{container.StateRunning, container.StateExited, container.StateCreated} {
		t.Run(string(state), func(t *testing.T) {
			a := testAttempt(t)
			_, err := a.RecordContainer("container-id")
			require.NoError(t, err)
			_, err = a.RequestStart(time.Now(), time.Minute)
			require.NoError(t, err)
			existing := ownedContainer(a)
			starts := 0
			runner := testRunner(t, func(req *http.Request) (*http.Response, error) {
				if req.Method == http.MethodGet {
					return dockerResponse(t, http.StatusOK, existing), nil
				}
				require.Equal(t, "/v1.52/containers/container-id/start", req.URL.Path)
				starts++
				existing.State.Status = state
				if state != container.StateCreated {
					existing.State.StartedAt = time.Now().UTC().Format(time.RFC3339Nano)
				}
				return dockerResponse(t, http.StatusServiceUnavailable, map[string]string{"message": "lost response"}), nil
			})
			err = runner.StartContainer(t.Context(), a)
			if state == container.StateCreated {
				require.ErrorIs(t, err, domain.ErrStartOutcomeUnknown)
			} else {
				require.NoError(t, err)
			}
			require.Equal(t, 1, starts)
		})
	}
}

func TestStartContainerReconcilesAfterCancellation(t *testing.T) {
	t.Parallel()
	a := testAttempt(t)
	_, err := a.RecordContainer("container-id")
	require.NoError(t, err)
	_, err = a.RequestStart(time.Now(), time.Minute)
	require.NoError(t, err)
	ctx, cancel := context.WithCancel(t.Context())
	defer cancel()
	existing := ownedContainer(a)
	runner := testRunner(t, func(req *http.Request) (*http.Response, error) {
		if req.Method == http.MethodPost {
			cancel()
			existing.State.Running = true
			return nil, context.Canceled
		}
		require.NoError(t, req.Context().Err())
		return dockerResponse(t, http.StatusOK, existing), nil
	})
	require.NoError(t, runner.StartContainer(ctx, a))
}

func TestStartContainerDoesNotRestartExitedContainer(t *testing.T) {
	t.Parallel()
	a := testAttempt(t)
	_, err := a.RecordContainer("container-id")
	require.NoError(t, err)
	_, err = a.RequestStart(time.Now(), time.Minute)
	require.NoError(t, err)
	existing := ownedContainer(a)
	existing.State.Status = container.StateExited
	existing.State.StartedAt = time.Now().UTC().Format(time.RFC3339Nano)
	runner := testRunner(t, func(req *http.Request) (*http.Response, error) {
		require.Equal(t, http.MethodGet, req.Method)
		return dockerResponse(t, http.StatusOK, existing), nil
	})
	require.NoError(t, runner.StartContainer(t.Context(), a))
}

func TestEnsureContainerReconcilesCanceledCreate(t *testing.T) {
	t.Parallel()
	a := testAttempt(t)
	existing := ownedContainer(a)
	ctx, cancel := context.WithCancel(t.Context())
	defer cancel()
	created := false
	runner := testRunner(t, func(req *http.Request) (*http.Response, error) {
		if req.Method == http.MethodPost {
			created = true
			cancel()
			return nil, context.Canceled
		}
		require.NoError(t, req.Context().Err())
		if !created {
			return dockerResponse(t, http.StatusNotFound, map[string]string{"message": "missing"}), nil
		}
		return dockerResponse(t, http.StatusOK, existing), nil
	})
	id, err := runner.EnsureContainer(ctx, a)
	require.NoError(t, err)
	require.Equal(t, existing.ID, id)
}

func TestStartContainerRequiresUnexpiredIntent(t *testing.T) {
	t.Parallel()
	a := testAttempt(t)
	runner := testRunner(t, func(req *http.Request) (*http.Response, error) {
		require.FailNow(t, "no Docker request expected", "%s", req.URL.Path)
		return nil, context.Canceled
	})
	require.ErrorIs(t, runner.StartContainer(t.Context(), a), errs.ErrConflict)
	_, err := a.RecordContainer("container-id")
	require.NoError(t, err)
	_, err = a.RequestStart(time.Now().Add(-time.Hour), time.Minute)
	require.NoError(t, err)
	require.ErrorIs(t, runner.StartContainer(t.Context(), a), context.DeadlineExceeded)
}
