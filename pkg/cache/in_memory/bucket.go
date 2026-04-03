package in_memory

import (
	"context"
	"sync"
	"time"
)

type entry struct {
	value     any
	expiresAt time.Time
}

type bucket struct {
	data map[string]entry

	mu   sync.RWMutex
	done chan struct{}
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

func (b *bucket) get(ctx context.Context, key string) (any, error) {
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	default:
	}

	b.mu.RLock()
	entry, ok := b.data[key]
	b.mu.RUnlock()

	if !ok {
		return nil, ErrNotFound
	}

	if time.Now().Before(entry.expiresAt) {
		return entry.value, nil
	}

	b.mu.Lock()
	defer b.mu.Unlock()

	entry, ok = b.data[key]
	if ok && time.Now().After(entry.expiresAt) {
		delete(b.data, key)
	}
	return nil, ErrNotFound
}

func (b *bucket) set(ctx context.Context, key string, value any, ttl time.Duration) error {
	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
	}

	entry := entry{
		value:     value,
		expiresAt: time.Now().Add(ttl),
	}

	b.mu.Lock()
	b.data[key] = entry
	b.mu.Unlock()

	return nil
}

func (b *bucket) delete(ctx context.Context, key string) error {
	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
	}

	b.mu.Lock()
	delete(b.data, key)
	b.mu.Unlock()

	return nil
}

func (b *bucket) cleanup() {
	b.mu.Lock()
	defer b.mu.Unlock()

	now := time.Now()
	for key, entry := range b.data {
		if entry.expiresAt.After(now) {
			delete(b.data, key)
		}
	}
}

func (b *bucket) close() {
	close(b.done)
}
