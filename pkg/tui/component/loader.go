// ABOUTME: Spinner animation component for showing progress
// ABOUTME: Cycles through animation frames; caller triggers Tick()

package component

import (
	"sync"

	"github.com/mauromedda/pi-coding-agent-go/pkg/tui"
)

var defaultFrames = []string{"⠋", "⠙", "⠹", "⠸", "⠼", "⠴", "⠦", "⠧", "⠇", "⠏"}

// Loader displays a spinner animation with an optional label.
type Loader struct {
	mu     sync.Mutex
	frames []string
	frame  int
	label  string
}

// NewLoader creates a Loader with the given label.
func NewLoader(label string) *Loader {
	return &Loader{
		frames: defaultFrames,
		label:  label,
	}
}

// SetLabel updates the spinner label.
func (l *Loader) SetLabel(label string) {
	l.mu.Lock()
	l.label = label
	l.mu.Unlock()
}

// Tick advances the spinner to the next frame.
func (l *Loader) Tick() {
	l.mu.Lock()
	l.frame = (l.frame + 1) % len(l.frames)
	l.mu.Unlock()
}

// Render draws the current spinner frame and label.
func (l *Loader) Render(out *tui.RenderBuffer, _ int) {
	l.mu.Lock()
	frame := l.frames[l.frame]
	label := l.label
	l.mu.Unlock()

	if label != "" {
		out.WriteLine(frame + " " + label)
	} else {
		out.WriteLine(frame)
	}
}

// Invalidate is a no-op for Loader.
func (l *Loader) Invalidate() {}
