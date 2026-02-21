// ABOUTME: Assistant message display component with streaming text and wrap caching
// ABOUTME: Uses strings.Builder for O(1) amortized appends; caches wrapped lines

package components

import (
	"strings"

	"github.com/mauromedda/pi-coding-agent-go/pkg/tui"
	"github.com/mauromedda/pi-coding-agent-go/pkg/tui/width"
)

// AssistantMessage renders an assistant's response.
type AssistantMessage struct {
	buf      strings.Builder
	thinking string
	dirty    bool

	// Cached wrapped output to avoid re-wrapping on every render.
	cachedLines []string
	cachedWidth int
}

// NewAssistantMessage creates an AssistantMessage component.
func NewAssistantMessage() *AssistantMessage {
	return &AssistantMessage{dirty: true}
}

// AppendText adds text to the message (for streaming).
func (a *AssistantMessage) AppendText(text string) {
	a.buf.WriteString(text)
	a.dirty = true
}

// SetThinking sets the thinking/reasoning text.
func (a *AssistantMessage) SetThinking(text string) {
	a.thinking = text
	a.dirty = true
}

// Render draws the assistant message.
func (a *AssistantMessage) Render(out *tui.RenderBuffer, w int) {
	if a.thinking != "" {
		out.WriteLine("\x1b[2m" + "thinking..." + "\x1b[0m")
	}

	if a.buf.Len() > 0 {
		// Re-wrap only when content or width changed
		if a.dirty || w != a.cachedWidth {
			a.cachedLines = width.WrapTextWithAnsi(a.buf.String(), w)
			a.cachedWidth = w
			a.dirty = false
		}
		out.WriteLines(a.cachedLines)
	}
}

// Invalidate marks for re-render.
func (a *AssistantMessage) Invalidate() {
	a.dirty = true
}
