// ABOUTME: Vim mode state machine for the Input component with normal, insert, and visual modes
// ABOUTME: Implements navigation (h/l/w/b/e/0/$), operators (d/c/x), and mode transitions (i/a/A/I/Esc)

package component

import (
	"unicode"

	"github.com/mauromedda/pi-coding-agent-go/pkg/tui/key"
)

// VimMode represents the current vim editing mode.
type VimMode int

const (
	VimInsert VimMode = iota // default: normal text editing
	VimNormal                // navigation + operators
	VimVisual                // visual selection (basic, not fully implemented)
)

// SetVimEnabled enables or disables vim mode on the input.
// Enabling starts in Normal mode; disabling resets to Insert mode.
func (inp *Input) SetVimEnabled(enabled bool) {
	inp.vimEnabled = enabled
	if enabled {
		inp.vimMode = VimNormal
	} else {
		inp.vimMode = VimInsert
	}
	inp.pendingOp = 0
}

// VimEnabled returns whether vim mode is active.
func (inp *Input) VimEnabled() bool {
	return inp.vimEnabled
}

// VimMode returns the current vim mode.
func (inp *Input) VimMode() VimMode {
	return inp.vimMode
}

// handleVimKey processes a key event in Normal or Visual mode.
func (inp *Input) handleVimKey(k key.Key) {
	if inp.vimMode == VimVisual {
		inp.handleVimVisualKey(k)
		return
	}
	inp.handleVimNormalKey(k)
}

// handleVimVisualKey handles keys in Visual mode.
// Currently only Esc is supported (returns to Normal).
func (inp *Input) handleVimVisualKey(k key.Key) {
	if k.Type == key.KeyEscape {
		inp.vimMode = VimNormal
		inp.pendingOp = 0
	}
}

// handleVimNormalKey handles keys in Normal mode.
func (inp *Input) handleVimNormalKey(k key.Key) {
	// Arrow keys work in normal mode too
	switch k.Type {
	case key.KeyLeft:
		inp.moveCursorLeft()
		return
	case key.KeyRight:
		inp.moveCursorRight()
		return
	case key.KeyEscape:
		inp.pendingOp = 0
		return
	case key.KeyRune:
		// handled below
	default:
		return
	}

	r := k.Rune

	// If there's a pending operator, handle the motion/repeat
	if inp.pendingOp != 0 {
		inp.handlePendingOp(r)
		return
	}

	switch r {
	// Navigation
	case 'h':
		inp.moveCursorLeft()
	case 'l':
		inp.moveCursorRight()
	case '0':
		inp.moveCursorHome()
	case '$':
		inp.moveCursorEnd()
	case 'w':
		inp.cursor = nextWordStart(inp.text, inp.cursor)
		inp.dirty = true
	case 'b':
		inp.cursor = prevWordStart(inp.text, inp.cursor)
		inp.dirty = true
	case 'e':
		inp.cursor = endOfWord(inp.text, inp.cursor)
		inp.dirty = true

	// Undo
	case 'u':
		inp.doUndo()

	// Insert transitions
	case 'i':
		inp.vimMode = VimInsert
	case 'a':
		if inp.cursor < len(inp.text) {
			inp.cursor++
		}
		inp.vimMode = VimInsert
		inp.dirty = true
	case 'A':
		inp.cursor = len(inp.text)
		inp.vimMode = VimInsert
		inp.dirty = true
	case 'I':
		inp.cursor = 0
		inp.vimMode = VimInsert
		inp.dirty = true

	// Delete char at cursor
	case 'x':
		inp.delete()

	// Pending operators
	case 'd', 'c':
		inp.pendingOp = r
	}
}

// handlePendingOp executes a pending operator (d or c) with the given motion key.
func (inp *Input) handlePendingOp(motion rune) {
	op := inp.pendingOp
	inp.pendingOp = 0

	switch {
	// dd / cc: operate on whole line
	case motion == op:
		inp.saveUndo()
		inp.text = inp.text[:0]
		inp.cursor = 0
		inp.dirty = true
		if op == 'c' {
			inp.vimMode = VimInsert
		}

	// dw / cw: delete to next word
	case motion == 'w':
		end := nextWordStart(inp.text, inp.cursor)
		if end > inp.cursor {
			inp.saveUndo()
			inp.text = append(inp.text[:inp.cursor], inp.text[end:]...)
			inp.dirty = true
		}
		if op == 'c' {
			inp.vimMode = VimInsert
		}

	default:
		// Unknown motion; discard pending operator
	}
}

// nextWordStart returns the position of the next word start after pos.
// Words are sequences of non-space characters separated by spaces.
func nextWordStart(text []rune, pos int) int {
	n := len(text)
	if pos >= n {
		return n
	}

	i := pos
	// Skip current word (non-spaces)
	for i < n && !unicode.IsSpace(text[i]) {
		i++
	}
	// Skip spaces
	for i < n && unicode.IsSpace(text[i]) {
		i++
	}
	return i
}

// prevWordStart returns the position of the previous word start before pos.
func prevWordStart(text []rune, pos int) int {
	if pos <= 0 {
		return 0
	}

	i := pos - 1
	// Skip spaces before cursor
	for i > 0 && unicode.IsSpace(text[i]) {
		i--
	}
	// Skip word chars backwards
	for i > 0 && !unicode.IsSpace(text[i-1]) {
		i--
	}
	return i
}

// endOfWord returns the position of the end of the current or next word.
func endOfWord(text []rune, pos int) int {
	n := len(text)
	if pos >= n-1 {
		return max(n-1, 0)
	}

	i := pos + 1
	// If on a space, skip spaces first
	for i < n && unicode.IsSpace(text[i]) {
		i++
	}
	// Move through word chars to end
	for i < n-1 && !unicode.IsSpace(text[i+1]) {
		i++
	}
	return i
}
