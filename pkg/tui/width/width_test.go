// ABOUTME: Tests for VisibleWidth and related width calculation utilities
// ABOUTME: Covers ASCII, Unicode, emoji, ANSI sequences, and cache behavior

package width

import "testing"

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
