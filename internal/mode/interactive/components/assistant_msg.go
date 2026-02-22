// ABOUTME: Assistant message display component with streaming text and wrap caching
// ABOUTME: Uses strings.Builder for O(1) amortized appends; braille spinner for thinking
// ABOUTME: Supports inline tool calls with Claude-style rendering

package components

import (
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/mauromedda/pi-coding-agent-go/pkg/tui"
	"github.com/mauromedda/pi-coding-agent-go/pkg/tui/theme"
	"github.com/mauromedda/pi-coding-agent-go/pkg/tui/width"
)

// brailleSpinnerFrames are the animation frames for the thinking indicator.
var brailleSpinnerFrames = []rune{'⠋', '⠙', '⠹', '⠸', '⠼', '⠴', '⠦', '⠧', '⠇', '⠏'}

// spinnerIntervalMs controls the spinner frame rate (80ms per frame).
const spinnerIntervalMs = 80

// SpinnerFrame returns the current braille spinner character.
// Uses wall-clock time; no background goroutine needed.
func SpinnerFrame() rune {
	idx := int(time.Now().UnixMilli()/spinnerIntervalMs) % len(brailleSpinnerFrames)
	return brailleSpinnerFrames[idx]
}

// AssistantMessage renders an assistant's response.
type AssistantMessage struct {
	mu       sync.Mutex
	buf      strings.Builder
	thinking string
	dirty    bool

	// Tool calls attached to this message
	toolCalls []*ToolCall

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
	a.mu.Lock()
	a.buf.WriteString(text)
	a.dirty = true
	a.mu.Unlock()
}

// SetThinking sets the thinking/reasoning text.
func (a *AssistantMessage) SetThinking(text string) {
	a.mu.Lock()
	a.thinking = text
	a.dirty = true
	a.mu.Unlock()
}

// AddToolCall adds a tool call to the message
func (a *AssistantMessage) AddToolCall(tc *ToolCall) {
	a.mu.Lock()
	a.toolCalls = append(a.toolCalls, tc)
	a.dirty = true
	a.mu.Unlock()
}

// GetToolCalls returns all tool calls
func (a *AssistantMessage) GetToolCalls() []*ToolCall {
	a.mu.Lock()
	defer a.mu.Unlock()
	return a.toolCalls
}

// ToggleToolCallExpand toggles expand state of a tool call
func (a *AssistantMessage) ToggleToolCallExpand(index int) {
	a.mu.Lock()
	if index >= 0 && index < len(a.toolCalls) {
		a.toolCalls[index].ToggleExpand()
		a.dirty = true
	}
	a.mu.Unlock()
}

// Render draws the assistant message with a blank spacer above.
// Snapshots text under lock, wraps outside lock to avoid blocking AppendText.
func (a *AssistantMessage) Render(out *tui.RenderBuffer, w int) {
	a.mu.Lock()
	thinking := a.thinking
	needsWrap := a.buf.Len() > 0 && (a.dirty || w != a.cachedWidth)
	var rawText string
	if needsWrap {
		rawText = a.buf.String()
		a.dirty = false
	}
	lines := a.cachedLines
	toolCalls := a.toolCalls
	a.mu.Unlock()

	if needsWrap {
		wrapped := width.WrapTextWithAnsi(rawText, w)
		a.mu.Lock()
		a.cachedLines = wrapped
		a.cachedWidth = w
		lines = wrapped
		a.mu.Unlock()
	}

	out.WriteLine("")

	if thinking != "" {
		p := theme.Current().Palette
		spinner := SpinnerFrame()
		out.WriteLine(fmt.Sprintf("%s%c\x1b[0m %s", p.Info.Code(), spinner, p.Dim.Apply("Thinking...")))
	}

	if len(lines) > 0 {
		out.WriteLines(lines)
	}

	// Render tool calls after text content
	for _, tc := range toolCalls {
		// Add some spacing before tool call
		out.WriteLine("")
		tc.Render(out, w)
		// Add spacing after tool call
		out.WriteLine("")
	}
}

// Invalidate marks for re-render.
func (a *AssistantMessage) Invalidate() {
	a.mu.Lock()
	a.dirty = true
	a.mu.Unlock()
}
