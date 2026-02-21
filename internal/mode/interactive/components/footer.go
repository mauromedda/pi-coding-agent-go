// ABOUTME: Two-line status bar component showing context info and right-aligned metadata
// ABOUTME: Line 1: project/branch info; Line 2: left content + padded right content

package components

import (
	"strings"

	"github.com/mauromedda/pi-coding-agent-go/pkg/tui"
	"github.com/mauromedda/pi-coding-agent-go/pkg/tui/width"
)

// Footer displays status information at the bottom of the screen.
// It renders two lines: line1 for context, line2 with left/right alignment.
type Footer struct {
	line1      string
	line2Left  string
	line2Right string
}

// NewFooter creates a Footer component.
func NewFooter() *Footer {
	return &Footer{}
}

// SetLine1 sets the first line content (e.g. project path, branch).
func (f *Footer) SetLine1(s string) {
	f.line1 = s
}

// SetLine2 sets the second line with left-aligned and right-aligned parts.
func (f *Footer) SetLine2(left, right string) {
	f.line2Left = left
	f.line2Right = right
}

// SetContent updates the footer text (backward compatibility).
// Sets line1 to the given content and clears line2.
func (f *Footer) SetContent(content string) {
	f.line1 = content
	f.line2Left = ""
	f.line2Right = ""
}

// Content returns the current line1 text (backward compatibility).
func (f *Footer) Content() string {
	return f.line1
}

// Render writes the two footer lines with dim styling.
func (f *Footer) Render(out *tui.RenderBuffer, w int) {
	// Line 1
	out.WriteLine("\x1b[2m" + f.line1 + "\x1b[0m")

	// Line 2: left + padding + right
	leftW := width.VisibleWidth(f.line2Left)
	rightW := width.VisibleWidth(f.line2Right)
	pad := w - leftW - rightW
	if pad < 1 {
		pad = 1
	}
	line2 := "\x1b[2m" + f.line2Left + strings.Repeat(" ", pad) + f.line2Right + "\x1b[0m"
	out.WriteLine(line2)
}

// Invalidate is a no-op for Footer.
func (f *Footer) Invalidate() {}
