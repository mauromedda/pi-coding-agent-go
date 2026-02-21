// ABOUTME: Tests for VisibleWidth and related width calculation utilities
// ABOUTME: Covers ASCII, Unicode, emoji, ANSI sequences, and cache behavior

package width

import (
	"strings"
	"testing"
)

func TestVisibleWidth(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name  string
		input string
		want  int
	}{
		{name: "empty string", input: "", want: 0},
		{name: "ascii", input: "hello", want: 5},
		{name: "ansi colored", input: "\x1b[31mred\x1b[0m", want: 3},
		{name: "cjk", input: "你好", want: 4},
		{name: "mixed", input: "hi\x1b[1m!\x1b[0m", want: 3},
		{name: "emoji", input: "👋", want: 2},
		{name: "only ansi", input: "\x1b[31m\x1b[0m", want: 0},
		{name: "tabs not plain ascii", input: "a\tb", want: 2},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got := VisibleWidth(tt.input)
			if got != tt.want {
				t.Errorf("VisibleWidth(%q) = %d, want %d", tt.input, got, tt.want)
			}
		})
	}
}

func TestIsPlainASCII(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name  string
		input string
		want  bool
	}{
		{name: "plain ascii", input: "hello world!", want: true},
		{name: "with escape", input: "hello\x1b[31m", want: false},
		{name: "with tab", input: "a\tb", want: false},
		{name: "with newline", input: "a\nb", want: false},
		{name: "empty", input: "", want: true},
		{name: "unicode", input: "café", want: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got := isPlainASCII(tt.input)
			if got != tt.want {
				t.Errorf("isPlainASCII(%q) = %v, want %v", tt.input, got, tt.want)
			}
		})
	}
}

func TestCache_EvictionOrder(t *testing.T) {
	t.Parallel()

	c := newCache(3)
	c.put("a", 1)
	c.put("b", 2)
	c.put("c", 3)

	// Access "a" to promote it; "b" becomes LRU.
	if v, ok := c.get("a"); !ok || v != 1 {
		t.Fatalf("get(a) = %d, %v; want 1, true", v, ok)
	}

	// Add "d"; should evict "b" (LRU).
	c.put("d", 4)

	if _, ok := c.get("b"); ok {
		t.Error("expected 'b' to be evicted")
	}
	if v, ok := c.get("a"); !ok || v != 1 {
		t.Errorf("get(a) = %d, %v; want 1, true", v, ok)
	}
	if v, ok := c.get("d"); !ok || v != 4 {
		t.Errorf("get(d) = %d, %v; want 4, true", v, ok)
	}
}

func BenchmarkVisibleWidth_ASCII(b *testing.B) {
	s := "This is a plain ASCII string for benchmarking"
	for b.Loop() {
		VisibleWidth(s)
	}
}

func BenchmarkVisibleWidth_ANSI(b *testing.B) {
	s := "\x1b[31;1mColored\x1b[0m and \x1b[4munderlined\x1b[0m text"
	for b.Loop() {
		VisibleWidth(s)
	}
}

func BenchmarkVisibleWidth_Unicode(b *testing.B) {
	s := "你好世界 Hello 🌍"
	for b.Loop() {
		VisibleWidth(s)
	}
}

func BenchmarkCache_PutGet(b *testing.B) {
	c := newCache(256)
	keys := make([]string, 512)
	for i := range keys {
		keys[i] = strings.Repeat("x", i+1)
	}
	for b.Loop() {
		for i, k := range keys {
			c.put(k, i)
		}
		for _, k := range keys {
			c.get(k)
		}
	}
}
