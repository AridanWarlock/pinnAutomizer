package indempotency

import (
	"context"
	"encoding/json"
	"sync"
	"testing"
	"time"

	"github.com/AridanWarlock/pinnAutomizer/pkg/core"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Keep the same JSON boundary as the real Redis adapter, so incompatible formats fail.
type jsonRedis struct {
	mu   sync.Mutex
	data map[string][]byte
}

func (r *jsonRedis) Get(ctx context.Context, key string, target any) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	return json.Unmarshal(r.data[key], target)
}
func (r *jsonRedis) Set(ctx context.Context, key string, value any, ttl time.Duration) error {
	data, err := json.Marshal(value)
	if err != nil {
		return err
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	r.data[key] = data
	return nil
}
func (r *jsonRedis) TryLock(ctx context.Context, key string, value any, ttl time.Duration) (bool, error) {
	data, err := json.Marshal(value)
	if err != nil {
		return false, err
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	if _, ok := r.data[key]; ok {
		return false, nil
	}
	r.data[key] = data
	return true, nil
}
func (r *jsonRedis) Delete(ctx context.Context, key string) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	delete(r.data, key)
	return nil
}

func TestStoreConcurrentPendingCanBeRead(t *testing.T) {
	store := NewStore(&jsonRedis{data: make(map[string][]byte)}, time.Hour, time.Minute)
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
				if err == nil && status != core.IdempotencyStatusPending {
					t.Errorf("unexpected status %s", status)
				}
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
	assert.Equal(t, 1, acquired)
	for err := range failures {
		require.NoError(t, err)
	}
}

func TestStoreCompletedResultRoundTrip(t *testing.T) {
	store := NewStore(&jsonRedis{data: make(map[string][]byte)}, time.Hour, time.Minute)
	key := core.IdempotencyKey("request")
	want := struct {
		Value string `json:"value"`
	}{Value: "accepted"}
	require.NoError(t, store.Set(context.Background(), key, core.IdempotencyStatusCompleted, want))
	var got struct {
		Value string `json:"value"`
	}
	status, err := store.Get(context.Background(), key, &got)
	require.NoError(t, err)
	assert.Equal(t, core.IdempotencyStatusCompleted, status)
	assert.Equal(t, want, got)
}
