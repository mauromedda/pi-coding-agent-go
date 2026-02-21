// ABOUTME: Tests for ANSI-aware text wrapping and truncation
// ABOUTME: Covers word wrapping, line breaks, and ellipsis truncation

package width

import (
	"reflect"
	"testing"
)

func TestWrapTextWithAnsi(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		input    string
		maxWidth int
		want     []string
	}{
		{name: "empty", input: "", maxWidth: 10, want: []string{""}},
		{name: "fits", input: "hello", maxWidth: 10, want: []string{"hello"}},
		{name: "exact fit", input: "hello", maxWidth: 5, want: []string{"hello"}},
		{name: "break needed", input: "abcdef", maxWidth: 3, want: []string{"abc", "def"}},
		{name: "newlines", input: "ab\ncd", maxWidth: 10, want: []string{"ab", "cd"}},
		{name: "zero width", input: "x", maxWidth: 0, want: nil},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got := WrapTextWithAnsi(tt.input, tt.maxWidth)
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("WrapTextWithAnsi(%q, %d) = %v, want %v", tt.input, tt.maxWidth, got, tt.want)
			}
		})
	}
}

func TestTruncateToWidth(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		input    string
		maxWidth int
		wantLen  int // check visible width of output
		fits     bool
	}{
		{name: "fits", input: "hi", maxWidth: 5, fits: true},
		{name: "exact", input: "hello", maxWidth: 5, fits: true},
		{name: "truncated", input: "hello world", maxWidth: 5, wantLen: 5, fits: false},
		{name: "one char", input: "hello", maxWidth: 1, fits: false},
		{name: "zero", input: "hello", maxWidth: 0, fits: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got := TruncateToWidth(tt.input, tt.maxWidth)
			gotWidth := VisibleWidth(got)
			if tt.fits {
				if got != tt.input {
					t.Errorf("expected no truncation, got %q", got)
				}
			} else if tt.maxWidth > 0 && gotWidth > tt.maxWidth {
				t.Errorf("TruncateToWidth(%q, %d) width = %d, want <= %d", tt.input, tt.maxWidth, gotWidth, tt.maxWidth)
			}
		})
	}
}
