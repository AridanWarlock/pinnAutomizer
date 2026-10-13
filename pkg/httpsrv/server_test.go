package httpsrv

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestShutdownAllowsActiveRequestToFinish(t *testing.T) {
	started := make(chan struct{})
	shuttingDown := make(chan struct{})
	release := make(chan struct{})
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		close(started)
		<-release
		_, _ = io.WriteString(w, "completed")
	}))
	defer server.Close()
	server.Config.RegisterOnShutdown(func() { close(shuttingDown) })
	requestDone := make(chan error, 1)
	go func() {
		response, err := server.Client().Get(server.URL)
		if err == nil {
			defer response.Body.Close()
			var body []byte
			body, err = io.ReadAll(response.Body)
			if err == nil && string(body) != "completed" {
				err = io.ErrUnexpectedEOF
			}
		}
		requestDone <- err
	}()
	<-started
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	done := make(chan error, 1)
	go func() { done <- shutdownServer(ctx, server.Config, time.Second) }()
	<-shuttingDown
	// Shutdown closes listeners before invoking these callbacks.
	close(release)
	require.NoError(t, <-requestDone)
	require.NoError(t, <-done)
}

func TestShutdownHasBoundedCleanupContext(t *testing.T) {
	server := &http.Server{}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	assert.NoError(t, shutdownServer(ctx, server, time.Second))
}
