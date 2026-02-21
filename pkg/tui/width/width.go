// ABOUTME: VisibleWidth computes display width of strings with grapheme-aware segmentation
// ABOUTME: Includes LRU cache for non-ASCII strings; fast path for pure ASCII

package width

import (
	"container/list"
	"sync"
	"unicode/utf8"

	"github.com/mattn/go-runewidth"
	"github.com/rivo/uniseg"
)

const cacheSize = 512

// lruEntry holds a cached width measurement.
type lruEntry struct {
	key   string
	value int
}

// cache is an O(1) LRU cache for non-ASCII string widths.
// Uses container/list for O(1) eviction and sync.RWMutex for concurrent reads.
type cache struct {
	mu    sync.RWMutex
	items map[string]*list.Element
	order *list.List
	size  int
}

func newCache(size int) *cache {
	return &cache{
		items: make(map[string]*list.Element, size),
		order: list.New(),
		size:  size,
	}
}

func (c *cache) get(key string) (int, bool) {
	c.mu.RLock()
	elem, ok := c.items[key]
	c.mu.RUnlock()
	if !ok {
		return 0, false
	}
	// Promote to front (requires write lock)
	c.mu.Lock()
	c.order.MoveToFront(elem)
	c.mu.Unlock()
	return elem.Value.(lruEntry).value, true
}

func (c *cache) put(key string, value int) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if _, ok := c.items[key]; ok {
		return
	}
	if c.order.Len() >= c.size {
		// Evict least recently used (back of list)
		back := c.order.Back()
		if back != nil {
			c.order.Remove(back)
			delete(c.items, back.Value.(lruEntry).key)
		}
	}
	elem := c.order.PushFront(lruEntry{key: key, value: value})
	c.items[key] = elem
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
	// Decode the first rune without allocating a []rune slice.
	r, _ := utf8.DecodeRuneInString(cluster)
	return runewidth.RuneWidth(r)
}
