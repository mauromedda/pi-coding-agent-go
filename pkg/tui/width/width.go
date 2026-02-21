// ABOUTME: VisibleWidth computes display width of strings with grapheme-aware segmentation
// ABOUTME: Includes LRU cache for non-ASCII strings; fast path for pure ASCII

package width

import (
	"sync"

	"github.com/mattn/go-runewidth"
	"github.com/rivo/uniseg"
)

const cacheSize = 512

// lruEntry holds a cached width measurement.
type lruEntry struct {
	key   string
	value int
}

// cache is a simple LRU cache for non-ASCII string widths.
type cache struct {
	mu      sync.Mutex
	entries []lruEntry
	index   map[string]int
	size    int
}

func newCache(size int) *cache {
	return &cache{
		entries: make([]lruEntry, 0, size),
		index:   make(map[string]int, size),
		size:    size,
	}
}

func (c *cache) get(key string) (int, bool) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if idx, ok := c.index[key]; ok {
		return c.entries[idx].value, true
	}
	return 0, false
}

func (c *cache) put(key string, value int) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if _, ok := c.index[key]; ok {
		return
	}
	if len(c.entries) >= c.size {
		// Evict oldest
		old := c.entries[0]
		delete(c.index, old.key)
		c.entries = c.entries[1:]
		// Re-index
		for i, e := range c.entries {
			c.index[e.key] = i
		}
	}
	c.index[key] = len(c.entries)
	c.entries = append(c.entries, lruEntry{key: key, value: value})
}

var widthCache = newCache(cacheSize)

// VisibleWidth returns the display width of s, accounting for ANSI escape
// sequences (which contribute zero width) and grapheme clusters (which may
// be wider than one cell for East Asian characters and emoji).
func VisibleWidth(s string) int {
	if s == "" {
		return 0
	}
	// Fast path: pure ASCII with no escape sequences
	if isPlainASCII(s) {
		return len(s)
	}
	// Check cache
	if w, ok := widthCache.get(s); ok {
		return w
	}
	w := computeWidth(s)
	widthCache.put(s, w)
	return w
}

// isPlainASCII returns true if s contains only printable ASCII (0x20-0x7E)
// with no escape sequences.
func isPlainASCII(s string) bool {
	for i := 0; i < len(s); i++ {
		b := s[i]
		if b < 0x20 || b > 0x7E {
			return false
		}
	}
	return true
}

// computeWidth measures the visible width by iterating grapheme clusters,
// skipping ANSI escape sequences.
func computeWidth(s string) int {
	stripped := StripANSI(s)
	w := 0
	state := -1
	for len(stripped) > 0 {
		cluster, rest, _, newState := uniseg.FirstGraphemeClusterInString(stripped, state)
		w += graphemeWidth(cluster)
		stripped = rest
		state = newState
	}
	return w
}

// graphemeWidth returns the display width of a single grapheme cluster.
func graphemeWidth(cluster string) int {
	if len(cluster) == 0 {
		return 0
	}
	// Use the first rune's width as baseline
	r := []rune(cluster)
	return runewidth.RuneWidth(r[0])
}
