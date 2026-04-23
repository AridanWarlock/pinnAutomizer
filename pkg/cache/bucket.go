package cache

import (
	"sync"
	"time"
)

type entry struct {
	value     any
	expiresAt time.Time
}

type bucket struct {
	data map[string]entry

	mu sync.RWMutex

	done   chan struct{}
	closed bool
}

func newBucket(cleanupInterval time.Duration) *bucket {
	b := &bucket{
		data: make(map[string]entry),
		mu:   sync.RWMutex{},
		done: make(chan struct{}),
	}

	go func() {
		ticker := time.NewTicker(cleanupInterval)
		for {
			select {
			case <-b.done:
				return
			case <-ticker.C:
				b.cleanup()
			}
		}
	}()

	return b
}

func (b *bucket) get(key string) (any, bool) {
	b.mu.RLock()
	entry, ok := b.data[key]
	b.mu.RUnlock()

	if !ok {
		return nil, false
	}

	if time.Now().Before(entry.expiresAt) {
		return entry.value, true
	}

	b.mu.Lock()
	defer b.mu.Unlock()

	entry, ok = b.data[key]
	if ok && time.Now().After(entry.expiresAt) {
		delete(b.data, key)
	}
	return nil, false
}

func (b *bucket) set(key string, value any, ttl time.Duration) {
	entry := entry{
		value:     value,
		expiresAt: time.Now().Add(ttl),
	}

	b.mu.Lock()
	b.data[key] = entry
	b.mu.Unlock()
}

func (b *bucket) delete(key string) {
	b.mu.Lock()
	delete(b.data, key)
	b.mu.Unlock()
}

func (b *bucket) cleanup() {
	b.mu.Lock()
	defer b.mu.Unlock()

	now := time.Now()
	for key, entry := range b.data {
		if entry.expiresAt.Before(now) {
			delete(b.data, key)
		}
	}
}

func (b *bucket) close() {
	if b.closed {
		return
	}

	b.mu.Lock()
	defer b.mu.Unlock()

	if b.closed {
		return
	}

	close(b.done)
	b.closed = true
}
