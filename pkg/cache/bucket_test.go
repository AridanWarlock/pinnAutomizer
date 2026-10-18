package cache

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestBucket_Cleanup(t *testing.T) {
	b := newBucket(time.Hour)
	defer b.close()
	b.mu.Lock()
	b.data["expired"] = entry{value: 1, expiresAt: time.Now().Add(-time.Second)}
	b.mu.Unlock()
	b.set("active", 2, time.Hour)
	b.cleanup()
	b.mu.RLock()
	defer b.mu.RUnlock()
	assert.NotContains(t, b.data, "expired")
	assert.Contains(t, b.data, "active")
}

func TestBucket_BackgroundCleanup(t *testing.T) {
	b := newBucket(time.Millisecond)
	defer b.close()
	b.mu.Lock()
	b.data["expired"] = entry{value: 1, expiresAt: time.Now().Add(-time.Second)}
	b.mu.Unlock()
	assert.Eventually(t, func() bool {
		b.mu.RLock()
		defer b.mu.RUnlock()
		_, ok := b.data["expired"]
		return !ok
	}, time.Second, time.Millisecond)
}

func TestBucketNonPositiveTTLDoesNotStore(t *testing.T) {
	b := newBucket(time.Hour)
	defer b.close()
	for _, ttl := range []time.Duration{0, -time.Second} {
		b.set("key", "old", time.Hour)
		b.set("key", "new", ttl)
		_, ok := b.get("key")
		assert.False(t, ok)
	}
}
