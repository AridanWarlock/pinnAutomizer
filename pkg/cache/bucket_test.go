package cache

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestBucket_Cleanup(t *testing.T) {
	cleanupInterval := 5 * time.Millisecond

	bucket := newBucket(cleanupInterval)
	bucket.set("key1", 1, time.Millisecond)

	// expires, but exists
	<-time.After(3 * time.Millisecond)
	assert.Len(t, bucket.data, 1)

	// cleaned up
	<-time.After(3 * time.Millisecond)
	assert.Len(t, bucket.data, 0)
}
