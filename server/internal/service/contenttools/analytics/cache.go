package analytics

import (
	"sync"
	"time"
)

// DefaultCacheTTL is the short TTL for aggregate reads (FR-16).
const DefaultCacheTTL = 60 * time.Second

type cacheEntry struct {
	value     any
	expiresAt time.Time
}

// AggregateCache is an in-process TTL cache keyed by scope strings.
type AggregateCache struct {
	mu    sync.RWMutex
	ttl   time.Duration
	items map[string]cacheEntry
}

// NewAggregateCache builds a cache with the given TTL (default 60s).
func NewAggregateCache(ttl time.Duration) *AggregateCache {
	if ttl <= 0 {
		ttl = DefaultCacheTTL
	}
	return &AggregateCache{ttl: ttl, items: map[string]cacheEntry{}}
}

// Get returns a cached value when present and not expired.
func (c *AggregateCache) Get(key string) (any, bool) {
	if c == nil {
		return nil, false
	}
	c.mu.RLock()
	defer c.mu.RUnlock()
	e, ok := c.items[key]
	if !ok || time.Now().After(e.expiresAt) {
		return nil, false
	}
	return e.value, true
}

// Set stores a value with the cache TTL.
func (c *AggregateCache) Set(key string, value any) {
	if c == nil {
		return
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	c.items[key] = cacheEntry{value: value, expiresAt: time.Now().Add(c.ttl)}
}

// InvalidatePrefix drops all keys with the given prefix (e.g. instance UUID).
func (c *AggregateCache) InvalidatePrefix(prefix string) {
	if c == nil || prefix == "" {
		return
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	for k := range c.items {
		if len(k) >= len(prefix) && k[:len(prefix)] == prefix {
			delete(c.items, k)
		}
	}
}

// InvalidateExact drops one key.
func (c *AggregateCache) InvalidateExact(key string) {
	if c == nil {
		return
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	delete(c.items, key)
}

var defaultCache = NewAggregateCache(DefaultCacheTTL)

// DefaultCache returns the process-wide aggregate cache.
func DefaultCache() *AggregateCache { return defaultCache }

// CacheKeyInstance builds a cache key for instance analytics.
func CacheKeyInstance(instanceID string) string {
	return "inst:" + instanceID
}

// CacheKeyItem builds a cache key for item rollup analytics.
func CacheKeyItem(itemID string) string {
	return "item:" + itemID
}

// CacheKeyCourse builds a cache key for course rollup analytics.
func CacheKeyCourse(courseID string) string {
	return "course:" + courseID
}

// InvalidateForInstance clears instance + course keys after state/reset writes.
func InvalidateForInstance(instanceID, courseID, itemID string) {
	c := DefaultCache()
	c.InvalidateExact(CacheKeyInstance(instanceID))
	if courseID != "" {
		c.InvalidateExact(CacheKeyCourse(courseID))
		c.InvalidatePrefix("course:" + courseID)
	}
	if itemID != "" {
		c.InvalidateExact(CacheKeyItem(itemID))
	}
}
