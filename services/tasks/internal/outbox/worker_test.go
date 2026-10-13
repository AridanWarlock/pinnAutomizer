package outbox

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/rs/zerolog"
	"github.com/stretchr/testify/require"
)

func TestWorkerCloseWaitsForTransactionCleanup(t *testing.T) {
	started := make(chan struct{})
	cleaning := make(chan struct{})
	release := make(chan struct{})
	var once sync.Once
	postgres := &MockPostgres{InTransactionFunc: func(ctx context.Context, fn func(context.Context) error) error {
		close(started)
		<-ctx.Done()
		close(cleaning)
		<-release
		return ctx.Err()
	}}
	worker := NewWorker(postgres, &MockWriter{}, zerolog.Nop())
	t.Cleanup(func() { once.Do(func() { close(release) }); worker.Close() })
	select {
	case <-started:
	case <-time.After(10 * time.Second):
		t.Fatal("worker did not start")
	}
	done := make(chan struct{})
	go func() { worker.Close(); close(done) }()
	select {
	case <-cleaning:
	case <-time.After(time.Second):
		t.Fatal("transaction was not canceled")
	}
	select {
	case <-done:
		t.Fatal("Close returned before transaction cleanup")
	default:
	}
	once.Do(func() { close(release) })
	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("Close did not return")
	}
	require.Len(t, postgres.InTransactionCalls(), 1)
}
