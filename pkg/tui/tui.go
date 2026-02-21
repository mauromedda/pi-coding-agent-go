// ABOUTME: TUI engine with differential rendering, focus management, and overlay compositing
// ABOUTME: Uses buffered channel for render coalescing; CSI 2026 synchronized output

package tui

import (
	"fmt"
	"strconv"
	"strings"
	"sync"

	"github.com/mauromedda/pi-coding-agent-go/pkg/tui/width"
)

// Writer is the minimal interface for terminal output.
type Writer interface {
	Write(p []byte) (n int, err error)
}

// TUI is the main rendering engine.
type TUI struct {
	container *Container
	writer    Writer
	width     int
	height    int

	mu            sync.Mutex
	previousLines []string
	overlays      []Overlay
	renderCh      chan struct{}
	stopCh        chan struct{}
	stopOnce      sync.Once
	running       bool

	// Relative rendering state
	rstate renderState
}

// New creates a new TUI engine writing to w with the given dimensions.
func New(w Writer, termWidth, termHeight int) *TUI {
	return &TUI{
		container: NewContainer(),
		writer:    w,
		width:     termWidth,
		height:    termHeight,
		renderCh:  make(chan struct{}, 1),
		stopCh:    make(chan struct{}),
		rstate:    renderState{firstRender: true},
	}
}

// Container returns the root container for adding components.
func (t *TUI) Container() *Container {
	return t.container
}

// SetSize updates the terminal dimensions and triggers a re-render.
func (t *TUI) SetSize(w, h int) {
	t.mu.Lock()
	t.width = w
	t.height = h
	t.previousLines = nil // Force full redraw
	// prevWidth mismatch will trigger full clear on next render
	t.mu.Unlock()
	t.container.Invalidate()
	t.RequestRender()
}

// PushOverlay adds a modal overlay on top of the content.
func (t *TUI) PushOverlay(o Overlay) {
	t.mu.Lock()
	t.overlays = append(t.overlays, o)
	t.mu.Unlock()
	t.RequestRender()
}

// PopOverlay removes the topmost overlay.
func (t *TUI) PopOverlay() {
	t.mu.Lock()
	if len(t.overlays) > 0 {
		t.overlays = t.overlays[:len(t.overlays)-1]
	}
	t.mu.Unlock()
	t.RequestRender()
}

// RequestRender signals that a render is needed. Multiple calls coalesce
// into a single render via a buffered channel of size 1.
func (t *TUI) RequestRender() {
	select {
	case t.renderCh <- struct{}{}:
	default: // Already pending; coalesced
	}
}

// Start begins the render loop in a goroutine. Call Stop to terminate.
func (t *TUI) Start() {
	t.mu.Lock()
	if t.running {
		t.mu.Unlock()
		return
	}
	t.running = true
	t.mu.Unlock()

	go t.renderLoop()
}

// Stop terminates the render loop. Safe to call multiple times.
func (t *TUI) Stop() {
	t.stopOnce.Do(func() {
		t.mu.Lock()
		if !t.running {
			t.mu.Unlock()
			return
		}
		t.running = false
		t.mu.Unlock()
		close(t.stopCh)
	})
}

// RenderOnce performs a single synchronous render. Useful for testing.
func (t *TUI) RenderOnce() {
	t.render()
}

func (t *TUI) renderLoop() {
	for {
		select {
		case <-t.stopCh:
			return
		case <-t.renderCh:
			t.render()
		}
	}
}

func (t *TUI) render() {
	t.mu.Lock()
	w := t.width
	h := t.height
	prevLines := t.previousLines
	rstate := t.rstate
	overlays := make([]Overlay, len(t.overlays))
	copy(overlays, t.overlays)
	t.mu.Unlock()

	if w <= 0 || h <= 0 {
		return
	}

	// Render main content
	buf := AcquireBuffer()
	defer ReleaseBuffer(buf)

	t.container.Render(buf, w)

	// Composite overlays on top
	compositeOverlays(buf, overlays, w, h)

	// Clamp to terminal height: keep bottom lines so editor+footer stay visible
	lines := buf.Lines
	clamped := len(lines) > h
	if clamped {
		lines = lines[len(lines)-h:]
	}

	// Detect clamp transition: force full redraw so diff engine stays consistent
	if clamped != rstate.prevClamped {
		prevLines = nil
		rstate.firstRender = true
		rstate.maxRendered = 0
	}
	rstate.prevClamped = clamped

	// Find cursor position and strip marker
	cursorRow, cursorCol := extractCursorPosition(lines)

	// Relative differential update
	output := relativeRender(&rstate, prevLines, lines, w)

	// Position cursor using relative movement
	if cursorRow >= 0 && cursorCol >= 0 {
		var curBuf strings.Builder
		var numBuf [20]byte
		moveCursor(&curBuf, numBuf[:], rstate.cursorRow, cursorRow)
		rstate.cursorRow = cursorRow
		curBuf.WriteString(fmt.Sprintf("\r\x1b[%dC", cursorCol))
		curBuf.WriteString("\x1b[?25h") // Show cursor
		output += curBuf.String()
	} else {
		output += "\x1b[?25l" // Hide cursor
	}

	// Write output atomically
	if output != "" {
		// CSI 2026 synchronized output: begin
		syncOutput := "\x1b[?2026h" + output + "\x1b[?2026l"
		_, _ = t.writer.Write([]byte(syncOutput))
	}

	// Save current lines for next diff, reusing the previous slice when possible.
	saved := prevLines
	if cap(saved) >= len(lines) {
		saved = saved[:len(lines)]
	} else {
		saved = make([]string, len(lines))
	}
	copy(saved, lines)
	t.mu.Lock()
	t.previousLines = saved
	t.rstate = rstate
	t.mu.Unlock()
}

// compositeOverlays renders overlays on top of the main buffer.
func compositeOverlays(buf *RenderBuffer, overlays []Overlay, w, h int) {
	for _, o := range overlays {
		overlayBuf := AcquireBuffer()
		ow := o.Width
		if ow <= 0 {
			ow = w
		}
		o.Component.Render(overlayBuf, ow)

		oh := overlayBuf.Len()
		if o.Height > 0 && oh > o.Height {
			oh = o.Height
		}

		// Calculate vertical position
		var startRow int
		switch o.Position {
		case OverlayCenter:
			startRow = (h - oh) / 2
		case OverlayTop:
			startRow = 0
		case OverlayBottom:
			startRow = h - oh
		}
		if startRow < 0 {
			startRow = 0
		}

		// Ensure buf has enough lines
		for buf.Len() < startRow+oh {
			buf.WriteLine("")
		}

		// Overlay lines
		for i := 0; i < oh && i < overlayBuf.Len(); i++ {
			row := startRow + i
			if row < len(buf.Lines) {
				buf.Lines[row] = overlayBuf.Lines[i]
			}
		}

		ReleaseBuffer(overlayBuf)
	}
}

// extractCursorPosition finds the CursorMarker in lines, removes it,
// and returns (row, col). Returns (-1, -1) if not found.
func extractCursorPosition(lines []string) (row, col int) {
	for i, line := range lines {
		idx := strings.Index(line, CursorMarker)
		if idx >= 0 {
			before := line[:idx]
			after := line[idx+len(CursorMarker):]
			lines[i] = before + after
			return i, width.VisibleWidth(before)
		}
	}
	return -1, -1
}

// renderState tracks cursor position across renders for relative movement.
type renderState struct {
	maxRendered int  // max lines ever rendered
	cursorRow   int  // cursor row (0-based, relative to our output region)
	firstRender bool // true until first render completes
	prevWidth   int  // detect width changes
	prevClamped bool // was previous frame clamped to terminal height?
}

// relativeRender generates ANSI commands using relative cursor movement
// instead of absolute positioning, so content scrolls like a chat.
func relativeRender(state *renderState, prev, curr []string, termWidth int) string {
	var b strings.Builder
	var numBuf [20]byte

	// Width change: full clear and re-render everything
	if state.prevWidth != 0 && state.prevWidth != termWidth {
		b.WriteString("\x1b[2J\x1b[H") // clear screen + home
		for i, line := range curr {
			if i > 0 {
				b.WriteString("\r\n")
			}
			b.WriteString(line)
		}
		state.cursorRow = len(curr) - 1
		if state.cursorRow < 0 {
			state.cursorRow = 0
		}
		state.maxRendered = len(curr)
		state.firstRender = false
		state.prevWidth = termWidth
		return b.String()
	}
	state.prevWidth = termWidth

	// First render: just output lines with \r\n
	if state.firstRender {
		for i, line := range curr {
			if i > 0 {
				b.WriteString("\r\n")
			}
			b.WriteString(line)
		}
		state.cursorRow = len(curr) - 1
		if state.cursorRow < 0 {
			state.cursorRow = 0
		}
		state.maxRendered = len(curr)
		state.firstRender = false
		return b.String()
	}

	// Find which lines changed and which are new
	commonLen := len(prev)
	if len(curr) < commonLen {
		commonLen = len(curr)
	}

	// Update changed lines using relative movement
	for i := 0; i < commonLen; i++ {
		if prev[i] == curr[i] {
			continue
		}
		// Move cursor to row i
		moveCursor(&b, numBuf[:], state.cursorRow, i)
		state.cursorRow = i
		b.WriteString("\r\x1b[2K") // carriage return + erase line
		b.WriteString(curr[i])
	}

	// Append new lines
	if len(curr) > len(prev) {
		// Move to the last rendered line
		moveCursor(&b, numBuf[:], state.cursorRow, len(prev)-1)
		state.cursorRow = len(prev) - 1
		if state.cursorRow < 0 {
			state.cursorRow = 0
		}

		for i := len(prev); i < len(curr); i++ {
			b.WriteString("\r\n")
			b.WriteString(curr[i])
			state.cursorRow = i
		}
	}

	// Clear excess lines if content shrank
	if len(curr) < state.maxRendered {
		for i := len(curr); i < state.maxRendered; i++ {
			moveCursor(&b, numBuf[:], state.cursorRow, i)
			state.cursorRow = i
			b.WriteString("\r\x1b[2K")
		}
		// Move back to last content line
		if len(curr) > 0 {
			moveCursor(&b, numBuf[:], state.cursorRow, len(curr)-1)
			state.cursorRow = len(curr) - 1
		}
		// Reset so we don't re-clear on next frame
		state.maxRendered = len(curr)
	}

	if len(curr) > state.maxRendered {
		state.maxRendered = len(curr)
	}

	return b.String()
}

// moveCursor emits relative cursor movement sequences to move from fromRow to toRow.
func moveCursor(b *strings.Builder, numBuf []byte, fromRow, toRow int) {
	if fromRow == toRow {
		return
	}
	delta := toRow - fromRow
	if delta < 0 {
		// Move up
		b.WriteString("\x1b[")
		b.Write(strconv.AppendInt(numBuf[:0], int64(-delta), 10))
		b.WriteByte('A')
	} else {
		// Move down
		b.WriteString("\x1b[")
		b.Write(strconv.AppendInt(numBuf[:0], int64(delta), 10))
		b.WriteByte('B')
	}
}
