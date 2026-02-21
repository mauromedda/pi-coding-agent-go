// ABOUTME: Single-line text display with ellipsis truncation
// ABOUTME: Truncates to terminal width when content exceeds available space

package component

import (
	"github.com/mauromedda/pi-coding-agent-go/pkg/tui"
	"github.com/mauromedda/pi-coding-agent-go/pkg/tui/width"
)

// TruncatedText renders a single line, truncating with ellipsis if needed.
type TruncatedText struct {
	content string
}

// NewTruncatedText creates a TruncatedText with the given content.
func NewTruncatedText(content string) *TruncatedText {
	return &TruncatedText{content: content}
}

// SetContent updates the text.
func (t *TruncatedText) SetContent(content string) {
	t.content = content
}

// Render writes the truncated line into the buffer.
func (t *TruncatedText) Render(out *tui.RenderBuffer, w int) {
	out.WriteLine(width.TruncateToWidth(t.content, w))
}

// Invalidate is a no-op for TruncatedText.
func (t *TruncatedText) Invalidate() {}
