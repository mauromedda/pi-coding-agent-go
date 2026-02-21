// ABOUTME: Static text display component for the TUI
// ABOUTME: Renders pre-defined lines with optional ANSI styling

package component

import "github.com/mauromedda/pi-coding-agent-go/pkg/tui"

// Text renders static text content.
type Text struct {
	content string
	lines   []string
	dirty   bool
}

// NewText creates a Text component with the given content.
func NewText(content string) *Text {
	return &Text{content: content, dirty: true}
}

// SetContent updates the displayed text.
func (t *Text) SetContent(content string) {
	t.content = content
	t.dirty = true
}

// Render writes the text lines into the buffer.
func (t *Text) Render(out *tui.RenderBuffer, width int) {
	if t.dirty {
		t.lines = splitLines(t.content)
		t.dirty = false
	}
	out.WriteLines(t.lines)
}

// Invalidate marks the component for re-render.
func (t *Text) Invalidate() {
	t.dirty = true
}

// splitLines splits a string by newlines.
func splitLines(s string) []string {
	if s == "" {
		return []string{""}
	}
	var lines []string
	start := 0
	for i := 0; i < len(s); i++ {
		if s[i] == '\n' {
			lines = append(lines, s[start:i])
			start = i + 1
		}
	}
	lines = append(lines, s[start:])
	return lines
}
