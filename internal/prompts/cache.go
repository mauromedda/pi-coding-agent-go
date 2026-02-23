// ABOUTME: SHA256-keyed cache for composed prompt output
// ABOUTME: Invalidated when version or variable values change

package prompts

import (
	"crypto/sha256"
	"fmt"
	"sort"
	"strings"
	"sync"
)

// Cache stores composed prompts keyed by a hash of version + variables.
type Cache struct {
	mu      sync.RWMutex
	entries map[string]string
}

// NewCache creates an empty cache.
func NewCache() *Cache {
	return &Cache{
		entries: make(map[string]string),
	}
}

// Get returns a cached prompt and true, or empty string and false.
func (c *Cache) Get(version string, vars map[string]string) (string, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	key := cacheKey(version, vars)
	val, ok := c.entries[key]
	return val, ok
}

// Set stores a composed prompt in the cache.
func (c *Cache) Set(version string, vars map[string]string, prompt string) {
	c.mu.Lock()
	defer c.mu.Unlock()

	key := cacheKey(version, vars)
	c.entries[key] = prompt
}

// Invalidate clears all cache entries.
func (c *Cache) Invalidate() {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.entries = make(map[string]string)
}

// Size returns the number of cached entries.
func (c *Cache) Size() int {
	c.mu.RLock()
	defer c.mu.RUnlock()

	return len(c.entries)
}

// cacheKey produces a SHA256 hex string from version + sorted variable pairs.
func cacheKey(version string, vars map[string]string) string {
	keys := make([]string, 0, len(vars))
	for k := range vars {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	var b strings.Builder
	b.WriteString(version)
	b.WriteByte('\x00')
	for _, k := range keys {
		b.WriteString(k)
		b.WriteByte('=')
		b.WriteString(vars[k])
		b.WriteByte('\x00')
	}

	h := sha256.Sum256([]byte(b.String()))
	return fmt.Sprintf("%x", h)
}
