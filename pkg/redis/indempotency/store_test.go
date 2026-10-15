package indempotency

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/AridanWarlock/pinnAutomizer/pkg/core"
)

func TestStoreConcurrentPendingCanBeRead(t *testing.T) {
	var held sync.Mutex
	pending := envelope{Status: core.IdempotencyStatusPending}
	redis := &MockRedis{
		TryLockFunc: func(_ context.Context, _ string, value any, ttl time.Duration) (bool, error) {
			assert.Equal(t, pending, value)
			assert.Equal(t, time.Minute, ttl)
			return held.TryLock(), nil
		},
		GetFunc: func(_ context.Context, _ string, target any) error {
			*target.(*envelope) = pending
			return nil
		},
		DeleteFunc: func(context.Context, string) error {
			held.Unlock()
			return nil
		},
	}
	store := NewStore(redis, time.Hour, time.Minute)
	key := core.IdempotencyKey("request")
	ctx := context.Background()
	var group sync.WaitGroup
	results := make(chan bool, 16)
	failures := make(chan error, 16)
	for range 16 {
		group.Go(func() {
			ok, err := store.TryLock(ctx, key)
			results <- ok
			if err == nil && !ok {
				var status core.IdempotencyStatus
				status, err = store.Get(ctx, key, nil)
				assert.Equal(t, core.IdempotencyStatusPending, status)
			}
			failures <- err
		})
	}
	group.Wait()
	close(results)
	close(failures)
	acquired := 0
	for ok := range results {
		if ok {
			acquired++
		}
	}
	require.Equal(t, 1, acquired)
	for err := range failures {
		require.NoError(t, err)
	}
	// The winner keeps ownership until every contender has completed.
	require.NoError(t, store.Delete(ctx, key))
	ok, err := store.TryLock(ctx, key)
	require.NoError(t, err)
	require.True(t, ok)
	require.NoError(t, store.Delete(ctx, key))
}

func TestStoreCompletedResultRoundTrip(t *testing.T) {
	var stored envelope
	redis := &MockRedis{
		SetFunc: func(_ context.Context, _ string, value any, _ time.Duration) error {
			stored = value.(envelope)
			return nil
		},
		GetFunc: func(_ context.Context, _ string, target any) error {
			*target.(*envelope) = stored
			return nil
		},
	}
	store := NewStore(redis, time.Hour, time.Minute)
	key := core.IdempotencyKey("request")
	want := struct {
		Value string `json:"value"`
	}{Value: "accepted"}
	require.NoError(t, store.Set(context.Background(), key, core.IdempotencyStatusCompleted, want))
	assert.JSONEq(t, `{"value":"accepted"}`, string(stored.Data))
	var got struct {
		Value string `json:"value"`
	}
	status, err := store.Get(context.Background(), key, &got)
	require.NoError(t, err)
	assert.Equal(t, core.IdempotencyStatusCompleted, status)
	assert.Equal(t, want, got)
}
