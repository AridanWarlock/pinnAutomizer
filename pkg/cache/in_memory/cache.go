package in_memory

import (
	"context"
	"slices"
	"sort"
	"strconv"
	"sync"
	"time"

	"github.com/cespare/xxhash"
)

const (
	DefaultBuckets         = 10
	DefaultReplicas        = 100
	DefaultTtl             = time.Minute
	DefaultCleanupInterval = 10 * time.Minute
)

var (
	DefaultHashFunc = xxhash.Sum64
)

type Cache struct {
	replicas int
	ring     []uint64
	nodes    map[uint64]int
	buckets  map[int]*bucket

	hashFunc func(key []byte) uint64
	mu       sync.RWMutex

	cleanupInterval time.Duration
}

func New(
	numBuckets int,
	replicas int,
	hashFunc func(key []byte) uint64,
	cleanupInterval time.Duration,
) *Cache {
	if replicas <= 0 {
		replicas = DefaultReplicas
	}

	var ring []uint64
	nodes := make(map[uint64]int)

	buckets := make(map[int]*bucket, numBuckets)
	for bucketID := range buckets {
		buckets[bucketID] = newBucket(cleanupInterval)

		for i := range replicas {
			replicaKey := strconv.Itoa(bucketID) + ":" + strconv.Itoa(i)

			hash := hashFunc([]byte(replicaKey))
			ring = append(ring, hash)

			nodes[hash] = bucketID
		}
	}

	slices.Sort(ring)

	return &Cache{
		replicas: replicas,
		ring:     make([]uint64, numBuckets),
		nodes:    nodes,
		buckets:  buckets,

		hashFunc: hashFunc,
		mu:       sync.RWMutex{},

		cleanupInterval: cleanupInterval,
	}
}

func (c *Cache) getBucket(key string) *bucket {
	c.mu.RLock()
	defer c.mu.RUnlock()

	hash := c.hashFunc([]byte(key))

	idx := sort.Search(len(c.ring), func(i int) bool {
		return c.ring[i] >= hash
	})
	if idx == len(c.ring) {
		idx = 0
	}

	bucketID := c.nodes[c.ring[idx]]
	return c.buckets[bucketID]
}

func (c *Cache) AddBucket() {
	c.mu.Lock()
	defer c.mu.Unlock()

	bucketID := len(c.ring)
	bucket := newBucket(c.cleanupInterval)
	c.buckets[bucketID] = bucket

	for i := range c.replicas {
		replicaKey := strconv.Itoa(bucketID) + ":" + strconv.Itoa(i)

		hash := c.hashFunc([]byte(replicaKey))
		c.ring = append(c.ring, hash)

		c.nodes[hash] = bucketID
	}

	slices.Sort(c.ring)
}

func (c *Cache) RemoveBucket(id int) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if id < 0 {
		return ErrIndexOutOfRange
	}

	if len(c.buckets) == 1 {
		return ErrRemovingLastBucket
	}

	newRing := make([]uint64, 0, len(c.ring))
	for _, h := range c.ring {
		if c.nodes[h] == id {
			delete(c.nodes, h)
			continue
		}

		newRing = append(newRing, h)
	}
	c.ring = newRing

	delete(c.buckets, id)

	return nil
}

func (c *Cache) Get(ctx context.Context, key string) (any, bool) {
	b := c.getBucket(key)
	v, err := b.get(ctx, key)

	if err != nil {
		return nil, false
	}
	return v, true
}

func (c *Cache) Set(ctx context.Context, key string, value any, ttl time.Duration) error {
	if ttl <= 0 {
		ttl = DefaultTtl
	}

	b := c.getBucket(key)
	return b.set(ctx, key, value, ttl)
}

func (c *Cache) Delete(ctx context.Context, key string) error {
	b := c.getBucket(key)
	return b.delete(ctx, key)
}

func (c *Cache) Close() {
	for _, b := range c.buckets {
		b.close()
	}
}
