package cache

import (
	"math/rand/v2"
	"strconv"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestCache_GetSetDelete(t *testing.T) {
	ttl := time.Millisecond

	cache := NewCache()
	defer cache.Close()

	cache.Set("key1", 1, ttl)
	cache.Set("key2", "string value", ttl)
	cache.Set("long ttl", nil, time.Minute)

	v, ok := cache.Get("key1")
	assert.True(t, ok)
	assert.Equal(t, 1, v.(int))

	v, ok = cache.Get("not exist")
	assert.False(t, ok)
	assert.Equal(t, nil, v)

	cache.Delete("key1")
	// idempotency
	cache.Delete("key1")

	v, ok = cache.Get("key1")
	assert.False(t, ok)
	assert.Equal(t, nil, nil)

	v, ok = cache.Get("key2")
	assert.True(t, ok)
	assert.Equal(t, "string value", v.(string))

	<-time.After(ttl)

	v, ok = cache.Get("key2")
	assert.False(t, ok)
	assert.Equal(t, nil, nil)
	v, ok = cache.Get("long ttl")
	assert.True(t, ok)
	assert.Equal(t, nil, nil)
}

func TestCache_GetSetDeleteInGoroutines(t *testing.T) {
	start := time.Now()

	ttl := 500 * time.Millisecond
	cache := NewCache()
	defer cache.Close()

	var wg sync.WaitGroup
	wg.Add(10_000)
	for range 10_000 {
		go func() {
			defer wg.Done()

			for range 100 {
				action := rand.IntN(1000)
				switch {
				case action < 100:
					cache.Delete(strconv.Itoa(action))
				case action < 400:
					cache.Set(strconv.Itoa(action%100), 42, ttl)
				default:
					cache.Get(strconv.Itoa(action % 100))
				}
			}
		}()
	}

	wg.Wait()
	end := time.Since(start)
	assert.Less(t, end, time.Second, "to slow cache")
}

func TestCache_AddRemoveBucket(t *testing.T) {
	cache := NewCache(
		WithNumBuckets(5),
	)
	defer cache.Close()

	for cache.numBuckets < 100 {
		cache.AddBucket()
	}

	cache.Set("key1", 1, time.Minute)
	v, ok := cache.Get("key1")
	assert.True(t, ok)
	assert.Equal(t, 1, v.(int))

	for range 99 {
		err := cache.RemoveRandomBucket()
		assert.NoError(t, err)
	}

	err := cache.RemoveRandomBucket()
	assert.ErrorIs(t, err, ErrRemovingLastBucket)

	cache.Set("key2", 2, time.Minute)
	v, ok = cache.Get("key2")
	assert.True(t, ok)
	assert.Equal(t, 2, v.(int))
}
