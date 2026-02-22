// ABOUTME: Assistant message display component with streaming text and wrap caching
// ABOUTME: Uses strings.Builder for O(1) amortized appends; braille spinner for thinking

package components

import (
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/mauromedda/pi-coding-agent-go/pkg/tui"
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
		spinner := SpinnerFrame()
		out.WriteLine(fmt.Sprintf("\x1b[36m%c\x1b[0m \x1b[2mThinking...\x1b[0m", spinner))
	}

	if len(lines) > 0 {
		out.WriteLines(lines)
	}
}

// Invalidate marks for re-render.
func (a *AssistantMessage) Invalidate() {
	a.mu.Lock()
	a.dirty = true
	a.mu.Unlock()
}
