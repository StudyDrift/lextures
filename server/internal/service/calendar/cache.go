package calendar

import (
	"strings"
	"sync"
	"time"
)

const defaultCacheTTL = 15 * time.Minute

type cacheEntry struct {
	body    []byte
	expires time.Time
}

// FeedCache is an in-process TTL cache for generated iCal bodies (plan 16.5; Redis in 17.5).
type FeedCache struct {
	mu    sync.RWMutex
	ttl   time.Duration
	items map[string]cacheEntry
}

func NewFeedCache() *FeedCache {
	return &FeedCache{
		ttl:   defaultCacheTTL,
		items: make(map[string]cacheEntry),
	}
}

func (c *FeedCache) Get(key string, now time.Time) ([]byte, bool) {
	c.mu.RLock()
	e, ok := c.items[key]
	c.mu.RUnlock()
	if !ok || now.After(e.expires) {
		return nil, false
	}
	return e.body, true
}

func (c *FeedCache) GetStale(key string) ([]byte, bool) {
	c.mu.RLock()
	e, ok := c.items[key]
	c.mu.RUnlock()
	if !ok {
		return nil, false
	}
	return e.body, true
}

func (c *FeedCache) Set(key string, body []byte, now time.Time) {
	c.mu.Lock()
	c.items[key] = cacheEntry{body: append([]byte(nil), body...), expires: now.Add(c.ttl)}
	c.mu.Unlock()
}

func (c *FeedCache) InvalidateUser(userID string) {
	c.mu.Lock()
	for k := range c.items {
		if k == "user:"+userID || strings.HasSuffix(k, ":"+userID) {
			delete(c.items, k)
		}
	}
	c.mu.Unlock()
}

func (c *FeedCache) InvalidateCourse(courseID string) {
	c.mu.Lock()
	mid := ":" + courseID + ":"
	for k := range c.items {
		if strings.Contains(k, mid) {
			delete(c.items, k)
		}
	}
	c.mu.Unlock()
}

// DefaultFeedCache is the process-wide calendar feed cache.
var DefaultFeedCache = NewFeedCache()
