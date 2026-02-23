// ABOUTME: Markdown renderer wrapper around glamour for terminal output
// ABOUTME: Caches rendered results keyed by content hash + width

package btea

import (
	"crypto/sha256"
	"fmt"
	"strings"

	"github.com/charmbracelet/glamour"
)

// MarkdownRenderer wraps glamour to render markdown with caching.
type MarkdownRenderer struct {
	cache map[string]string // "hash:width" -> rendered
}

// NewMarkdownRenderer creates a MarkdownRenderer with an empty cache.
func NewMarkdownRenderer() *MarkdownRenderer {
	return &MarkdownRenderer{
		cache: make(map[string]string),
	}
}

// Render returns the terminal-styled rendering of the given markdown.
// Results are cached by content hash and width.
func (r *MarkdownRenderer) Render(md string, width int) string {
	if md == "" {
		return ""
	}

	key := cacheKey(md, width)
	if cached, ok := r.cache[key]; ok {
		return cached
	}

	renderer, err := glamour.NewTermRenderer(
		glamour.WithAutoStyle(),
		glamour.WithWordWrap(width),
	)
	if err != nil {
		// Fallback: return raw text
		return md
	}

	rendered, err := renderer.Render(md)
	if err != nil {
		return md
	}

	// Trim trailing whitespace that glamour adds
	rendered = strings.TrimRight(rendered, "\n ")

	r.cache[key] = rendered
	return rendered
}

// cacheKey produces a string key from content hash and width.
func cacheKey(content string, width int) string {
	h := sha256.Sum256([]byte(content))
	return fmt.Sprintf("%x:%d", h[:8], width)
}
