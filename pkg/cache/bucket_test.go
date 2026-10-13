package cache

import (
	"github.com/stretchr/testify/assert"
	"testing"
	"time"
)

func TestBucket_Cleanup(t *testing.T) {
	b := newBucket(time.Hour)
	defer b.close()
	b.set("expired", 1, -time.Second)
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
	b.set("expired", 1, -time.Second)
	assert.Eventually(t, func() bool {
		b.mu.RLock()
		defer b.mu.RUnlock()
		_, ok := b.data["expired"]
		return !ok
	}, time.Second, time.Millisecond)
}
