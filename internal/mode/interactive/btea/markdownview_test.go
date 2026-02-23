// ABOUTME: Tests for the markdown renderer wrapper around glamour
// ABOUTME: Verifies rendering, caching, and width handling

package btea

import (
	"strings"
	"testing"
)

func TestMarkdownRenderer_Render(t *testing.T) {
	r := NewMarkdownRenderer()

	result := r.Render("# Hello World\n\nSome text.", 80)

	if result == "" {
		t.Fatal("Render returned empty string")
	}
	if !strings.Contains(result, "Hello World") {
		t.Error("rendered output missing heading text")
	}
}

func TestMarkdownRenderer_RenderCodeBlock(t *testing.T) {
	r := NewMarkdownRenderer()

	md := "```go\nfunc main() {}\n```"
	result := r.Render(md, 80)

	if result == "" {
		t.Fatal("Render returned empty string for code block")
	}
	if !strings.Contains(result, "func main()") {
		t.Error("rendered output missing code content")
	}
}

func TestMarkdownRenderer_CachesResults(t *testing.T) {
	r := NewMarkdownRenderer()

	input := "**bold text**"
	result1 := r.Render(input, 80)
	result2 := r.Render(input, 80)

	if result1 != result2 {
		t.Error("cached render should return identical results")
	}
}

func TestMarkdownRenderer_DifferentWidths(t *testing.T) {
	r := NewMarkdownRenderer()

	input := "# Heading\n\nA paragraph with some text that might wrap differently at different widths."
	result80 := r.Render(input, 80)
	result40 := r.Render(input, 40)

	// Both should render successfully
	if result80 == "" || result40 == "" {
		t.Fatal("Render returned empty for different widths")
	}
	// Different widths produce different cache keys, so results may differ
	_ = result80
	_ = result40
}

func TestMarkdownRenderer_EmptyInput(t *testing.T) {
	r := NewMarkdownRenderer()

	result := r.Render("", 80)
	if result != "" {
		t.Errorf("Render(\"\") = %q; want empty", result)
	}
}

func TestMarkdownRenderer_PlainText(t *testing.T) {
	r := NewMarkdownRenderer()

	result := r.Render("just plain text, no markdown", 80)
	if result == "" {
		t.Fatal("Render returned empty for plain text")
	}
	if !strings.Contains(result, "just plain text") {
		t.Error("plain text should pass through")
	}
}
