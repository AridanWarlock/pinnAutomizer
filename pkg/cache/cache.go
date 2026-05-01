package cache

import (
	"math/rand"
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

type Option func(c *Cache)

func WithReplicas(replicas int) Option {
	return func(c *Cache) {
		c.replicas = replicas
	}
}

func WithNumBuckets(numBuckets int) Option {
	return func(c *Cache) {
		c.numBuckets = numBuckets
	}
}

func WithHashFunc(hashFunc func(key []byte) uint64) Option {
	return func(c *Cache) {
		c.hashFunc = hashFunc
	}
}

func WithCleanupInterval(cleanupInterval time.Duration) Option {
	return func(c *Cache) {
		c.cleanupInterval = cleanupInterval
	}
}

type Cache struct {
	numBuckets int
	replicas   int
	ring       []uint64
	nodes      map[uint64]*bucket

	hashFunc func(key []byte) uint64
	mu       sync.RWMutex

	cleanupInterval time.Duration
}

func NewCache(
	opts ...Option,
) *Cache {
	c := &Cache{
		mu: sync.RWMutex{},
	}
	for _, opt := range opts {
		opt(c)
	}

	if c.numBuckets <= 1 {
		c.numBuckets = DefaultBuckets
	}
	if c.replicas <= 0 {
		c.replicas = DefaultReplicas
	}
	if c.hashFunc == nil {
		c.hashFunc = DefaultHashFunc
	}
	if c.cleanupInterval <= 0 {
		c.cleanupInterval = DefaultCleanupInterval
	}

	var ring []uint64
	nodes := make(map[uint64]*bucket)

	for bucketID := range c.numBuckets {
		bucket := newBucket(c.cleanupInterval)

		for i := range c.replicas {
			replicaKey := strconv.Itoa(bucketID) + ":" + strconv.Itoa(i)

			hash := c.hashFunc([]byte(replicaKey))
			ring = append(ring, hash)

			nodes[hash] = bucket
		}
	}

	slices.Sort(ring)

	c.ring = ring
	c.nodes = nodes

	return c
}

func (c *Cache) getBucket(key string) *bucket {
	c.mu.RLock()
	defer c.mu.RUnlock()

	hash := c.hashFunc([]byte(key))
	nodeIdx := c.getNodeIdx(hash)

	return c.nodes[nodeIdx]
}

func (c *Cache) getNodeIdx(hash uint64) uint64 {
	idx := sort.Search(len(c.ring), func(i int) bool {
		return c.ring[i] >= hash
	})
	idx = idx % len(c.ring)

	nodeIdx := c.ring[idx]
	return nodeIdx
}

func (c *Cache) AddBucket() {
	c.mu.Lock()
	defer c.mu.Unlock()

	bucket := newBucket(c.cleanupInterval)
	for i := range c.replicas {
		replicaKey := strconv.Itoa(c.numBuckets) + ":" + strconv.Itoa(i)

		hash := c.hashFunc([]byte(replicaKey))
		c.ring = append(c.ring, hash)

		c.nodes[hash] = bucket
	}

	slices.Sort(c.ring)
	c.numBuckets++
}

func (c *Cache) RemoveRandomBucket() error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.numBuckets == 1 {
		return ErrRemovingLastBucket
	}

	nodeIdx := c.getNodeIdx(rand.Uint64())
	bucket := c.nodes[nodeIdx]

	newRing := make([]uint64, 0, len(c.ring))
	for _, h := range c.ring {
		if c.nodes[h] == bucket {
			delete(c.nodes, h)
			continue
		}

		newRing = append(newRing, h)
	}
	c.ring = newRing
	c.numBuckets--

	bucket.close()

	return nil
}

func (c *Cache) Get(key string) (any, bool) {
	b := c.getBucket(key)
	return b.get(key)
}

func (c *Cache) Set(key string, value any, ttl time.Duration) {
	if ttl <= 0 {
		ttl = DefaultTtl
	}

	b := c.getBucket(key)
	b.set(key, value, ttl)
}

func (c *Cache) Delete(key string) {
	b := c.getBucket(key)
	b.delete(key)
}

func (c *Cache) Close() {
	for _, bucket := range c.nodes {
		bucket.close()
	}
}
