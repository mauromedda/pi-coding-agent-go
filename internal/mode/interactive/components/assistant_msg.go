// ABOUTME: Assistant message display component with markdown and tool calls
// ABOUTME: Renders streaming text content and tool execution results

package components

import (
	"github.com/mauromedda/pi-coding-agent-go/pkg/tui"
	"github.com/mauromedda/pi-coding-agent-go/pkg/tui/width"
)

// AssistantMessage renders an assistant's response.
type AssistantMessage struct {
	text     string
	thinking string
	dirty    bool
}

// NewAssistantMessage creates an AssistantMessage component.
func NewAssistantMessage() *AssistantMessage {
	return &AssistantMessage{dirty: true}
}

// AppendText adds text to the message (for streaming).
func (a *AssistantMessage) AppendText(text string) {
	a.text += text
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

	if a.text != "" {
		lines := width.WrapTextWithAnsi(a.text, w)
		out.WriteLines(lines)
	}
}

// Invalidate marks for re-render.
func (a *AssistantMessage) Invalidate() {
	a.dirty = true
}
