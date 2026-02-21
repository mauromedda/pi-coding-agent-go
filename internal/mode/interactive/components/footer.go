// ABOUTME: Two-line status bar component showing context info and right-aligned metadata
// ABOUTME: Line 1: project/branch info; Line 2: left content + padded right content

package components

import (
	"fmt"
	"strings"
	"sync"

	"github.com/mauromedda/pi-coding-agent-go/internal/config"
	"github.com/mauromedda/pi-coding-agent-go/pkg/tui"
	"github.com/mauromedda/pi-coding-agent-go/pkg/tui/width"
)

// Footer displays status information at the bottom of the screen.
// It renders two lines: line1 for context, line2 with left/right alignment.
type Footer struct {
	line1      string
	line2Left  string
	line2Right string
	modeLabel  string
	contextPct int
	thinking   config.ThinkingLevel
	model      string
	mu         sync.Mutex
}

// NewFooter creates a Footer component.
func NewFooter() *Footer {
	return &Footer{
		thinking: config.ThinkingOff,
	}
}

// SetLine1 sets the first line content (e.g. project path, branch).
func (f *Footer) SetLine1(s string) {
	f.mu.Lock()
	f.line1 = s
	f.mu.Unlock()
}

// SetLine2 sets the second line with left-aligned and right-aligned parts.
func (f *Footer) SetLine2(left, right string) {
	f.mu.Lock()
	f.line2Left = left
	f.line2Right = right
	f.mu.Unlock()
}

// SetContent updates the footer text (backward compatibility).
// Sets line1 to the given content and clears line2.
func (f *Footer) SetContent(content string) {
	f.mu.Lock()
	f.line1 = content
	f.line2Left = ""
	f.line2Right = ""
	f.mu.Unlock()
}

// Content returns the current line1 text (backward compatibility).
func (f *Footer) Content() string {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.line1
}

// SetThinkingLevel sets the thinking level for display
func (f *Footer) SetThinkingLevel(level config.ThinkingLevel) {
	f.mu.Lock()
	f.thinking = level
	f.mu.Unlock()
}

// SetModel sets the model name for display
func (f *Footer) SetModel(name string) {
	f.mu.Lock()
	f.model = name
	f.mu.Unlock()
}

// SetContextPct sets the context window occupation percentage for display.
func (f *Footer) SetContextPct(pct int) {
	f.mu.Lock()
	f.contextPct = pct
	f.mu.Unlock()
}

// SetModeLabel sets the Plan/Edit mode indicator for display.
func (f *Footer) SetModeLabel(label string) {
	f.mu.Lock()
	f.modeLabel = label
	f.mu.Unlock()
}

// Render writes the two footer lines with dim styling.
func (f *Footer) Render(out *tui.RenderBuffer, w int) {
	f.mu.Lock()
	line1 := f.line1
	line2Left := f.line2Left
	line2Right := f.line2Right
	modeLabel := f.modeLabel
	contextPct := f.contextPct
	thinking := f.thinking
	model := f.model
	f.mu.Unlock()

	// Line 1
	out.WriteLine("\x1b[2m" + line1 + "\x1b[0m")

	// Build line 2 with optional mode label and thinking level
	line2 := line2Left
	if modeLabel != "" {
		line2 += fmt.Sprintf(" \x1b[33m%s\x1b[0m", modeLabel) // yellow for mode
	}
	if contextPct > 0 {
		ctxColor := "\x1b[90m" // dim for < 60%
		if contextPct >= 80 {
			ctxColor = "\x1b[31m" // red
		} else if contextPct >= 60 {
			ctxColor = "\x1b[33m" // yellow
		}
		line2 += fmt.Sprintf(" %sctx %d%%\x1b[0m", ctxColor, contextPct)
	}
	if thinking != config.ThinkingOff {
		thinkingColor := "\x1b[36m" // cyan for thinking levels
		thinkingStr := thinking.String()
		line2 += fmt.Sprintf(" \x1b[90m[%s]\x1b[0m", fmt.Sprintf("%s\x1b[36m%s\x1b[0m", thinkingColor, thinkingStr))
	}
	if model != "" {
		line2 += fmt.Sprintf(" \x1b[90m(%s)\x1b[0m", model)
	}

	leftW := width.VisibleWidth(line2)
	rightW := width.VisibleWidth(line2Right)
	pad := max(w-leftW-rightW, 1)
	fullLine2 := "\x1b[2m" + line2 + strings.Repeat(" ", pad) + line2Right + "\x1b[0m"
	out.WriteLine(fullLine2)
}

// Invalidate is a no-op for Footer.
func (f *Footer) Invalidate() {}
