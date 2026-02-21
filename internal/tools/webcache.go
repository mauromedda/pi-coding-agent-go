// ABOUTME: In-memory LRU cache with TTL for web fetch results
// ABOUTME: Thread-safe; max 100 entries; 15-minute expiry

package tools

import (
	"sync"
	"time"
)

const (
	cacheMaxEntries = 100
	cacheTTL        = 15 * time.Minute
)

type cacheEntry struct {
	value   string
	created time.Time
}

// webCache is a simple TTL cache for web fetch results.
type webCache struct {
	mu      sync.Mutex
	entries map[string]cacheEntry
}

func newWebCache() *webCache {
	return &webCache{entries: make(map[string]cacheEntry)}
}

func (c *webCache) Get(key string) (string, bool) {
	c.mu.Lock()
	defer c.mu.Unlock()

	entry, ok := c.entries[key]
	if !ok {
		return "", false
	}
	if time.Since(entry.created) > cacheTTL {
		delete(c.entries, key)
		return "", false
	}
	return entry.value, true
}

func (c *webCache) Set(key, value string) {
	c.mu.Lock()
	defer c.mu.Unlock()

	// Evict oldest if at capacity
	if len(c.entries) >= cacheMaxEntries {
		var oldest string
		var oldestTime time.Time
		for k, v := range c.entries {
			if oldest == "" || v.created.Before(oldestTime) {
				oldest = k
				oldestTime = v.created
			}
		}
		delete(c.entries, oldest)
	}

	c.entries[key] = cacheEntry{value: value, created: time.Now()}
}
