// ABOUTME: Single-line text input component with cursor, undo/redo, and kill ring
// ABOUTME: Supports horizontal scrolling, placeholder text, and Emacs-style keybindings

package component

import (
	"strings"

	"github.com/mauromedda/pi-coding-agent-go/pkg/tui"
	"github.com/mauromedda/pi-coding-agent-go/pkg/tui/internal/killring"
	"github.com/mauromedda/pi-coding-agent-go/pkg/tui/internal/undo"
	"github.com/mauromedda/pi-coding-agent-go/pkg/tui/key"
	"github.com/mauromedda/pi-coding-agent-go/pkg/tui/width"
)

const inputUndoDepth = 100

// inputState captures the text and cursor position for undo/redo.
type inputState struct {
	text   []rune
	cursor int
}

// Input is a single-line text input with cursor tracking, undo/redo, and kill ring.
type Input struct {
	text        []rune
	cursor      int
	placeholder string
	focused     bool
	dirty       bool
	scrollOff   int
	ring        *killring.KillRing
	undoStack   *undo.Stack[inputState]

	// Vim mode fields
	vimEnabled bool
	vimMode    VimMode
	pendingOp  rune // 0 = none, 'd' = delete, 'c' = change
}

// NewInput creates a new empty Input component.
func NewInput() *Input {
	return &Input{
		text:      make([]rune, 0, 64),
		ring:      killring.New(),
		undoStack: undo.New[inputState](inputUndoDepth),
		dirty:     true,
	}
}

// Text returns the current input text.
func (inp *Input) Text() string {
	return string(inp.text)
}

// SetText replaces the input text and moves cursor to the end.
func (inp *Input) SetText(s string) {
	inp.saveUndo()
	inp.text = []rune(s)
	inp.cursor = len(inp.text)
	inp.dirty = true
}

// CursorPos returns the cursor position in runes.
func (inp *Input) CursorPos() int {
	return inp.cursor
}

// SetPlaceholder sets the placeholder text shown when the input is empty.
func (inp *Input) SetPlaceholder(p string) {
	inp.placeholder = p
	inp.dirty = true
}

// SetFocused sets the focus state.
func (inp *Input) SetFocused(focused bool) {
	inp.focused = focused
	inp.dirty = true
}

// IsFocused returns the focus state.
func (inp *Input) IsFocused() bool {
	return inp.focused
}

// Invalidate marks the component for re-render.
func (inp *Input) Invalidate() {
	inp.dirty = true
}

// HandleInput processes raw terminal input data.
func (inp *Input) HandleInput(data string) {
	k := key.ParseKey(data)

	// Vim mode: in Normal or Visual mode, delegate to vim handler
	if inp.vimEnabled && inp.vimMode != VimInsert {
		inp.handleVimKey(k)
		return
	}

	// Vim mode: in Insert mode, Esc returns to Normal
	if inp.vimEnabled && k.Type == key.KeyEscape {
		inp.vimMode = VimNormal
		return
	}

	switch k.Type {
	case key.KeyRune:
		inp.insertRune(k.Rune)
	case key.KeyBackspace:
		inp.backspace()
	case key.KeyDelete:
		inp.delete()
	case key.KeyLeft:
		inp.moveCursorLeft()
	case key.KeyRight:
		inp.moveCursorRight()
	case key.KeyHome:
		inp.moveCursorHome()
	case key.KeyEnd:
		inp.moveCursorEnd()
	case key.KeyUnknown:
		inp.handleControlByte(data)
	default:
		inp.handleControlByte(data)
	}
}

func (inp *Input) handleControlByte(data string) {
	if len(data) != 1 {
		return
	}
	b := data[0]
	switch b {
	case 0x01: // Ctrl+A = home
		inp.moveCursorHome()
	case 0x05: // Ctrl+E = end
		inp.moveCursorEnd()
	case 0x0b: // Ctrl+K = kill to end of line
		inp.killToEnd()
	case 0x19: // Ctrl+Y = yank
		inp.yank()
	case 0x1a: // Ctrl+Z = undo
		inp.doUndo()
	case 0x17: // Ctrl+W = delete word backward
		inp.deleteWordBackward()
	}
}

func (inp *Input) insertRune(r rune) {
	inp.saveUndo()
	inp.text = append(inp.text, 0)
	copy(inp.text[inp.cursor+1:], inp.text[inp.cursor:])
	inp.text[inp.cursor] = r
	inp.cursor++
	inp.dirty = true
}

func (inp *Input) backspace() {
	if inp.cursor == 0 {
		return
	}
	inp.saveUndo()
	inp.text = append(inp.text[:inp.cursor-1], inp.text[inp.cursor:]...)
	inp.cursor--
	inp.dirty = true
}

func (inp *Input) delete() {
	if inp.cursor >= len(inp.text) {
		return
	}
	inp.saveUndo()
	inp.text = append(inp.text[:inp.cursor], inp.text[inp.cursor+1:]...)
	inp.dirty = true
}

func (inp *Input) moveCursorLeft() {
	if inp.cursor > 0 {
		inp.cursor--
		inp.dirty = true
	}
}

func (inp *Input) moveCursorRight() {
	if inp.cursor < len(inp.text) {
		inp.cursor++
		inp.dirty = true
	}
}

func (inp *Input) moveCursorHome() {
	inp.cursor = 0
	inp.dirty = true
}

func (inp *Input) moveCursorEnd() {
	inp.cursor = len(inp.text)
	inp.dirty = true
}

func (inp *Input) killToEnd() {
	if inp.cursor >= len(inp.text) {
		return
	}
	inp.saveUndo()
	killed := string(inp.text[inp.cursor:])
	inp.ring.Push(killed)
	inp.text = inp.text[:inp.cursor]
	inp.dirty = true
}

func (inp *Input) yank() {
	yanked := inp.ring.Yank()
	if yanked == "" {
		return
	}
	inp.saveUndo()
	runes := []rune(yanked)
	newText := make([]rune, 0, len(inp.text)+len(runes))
	newText = append(newText, inp.text[:inp.cursor]...)
	newText = append(newText, runes...)
	newText = append(newText, inp.text[inp.cursor:]...)
	inp.text = newText
	inp.cursor += len(runes)
	inp.dirty = true
}

func (inp *Input) doUndo() {
	state, ok := inp.undoStack.Undo()
	if !ok {
		return
	}
	inp.text = state.text
	inp.cursor = state.cursor
	inp.dirty = true
}

func (inp *Input) deleteWordBackward() {
	if inp.cursor == 0 {
		return
	}
	inp.saveUndo()
	pos := inp.cursor - 1
	// Skip spaces
	for pos > 0 && inp.text[pos] == ' ' {
		pos--
	}
	// Skip non-spaces (word chars)
	for pos > 0 && inp.text[pos-1] != ' ' {
		pos--
	}
	deleted := string(inp.text[pos:inp.cursor])
	inp.ring.Push(deleted)
	inp.text = append(inp.text[:pos], inp.text[inp.cursor:]...)
	inp.cursor = pos
	inp.dirty = true
}

func (inp *Input) saveUndo() {
	state := inputState{
		text:   make([]rune, len(inp.text)),
		cursor: inp.cursor,
	}
	copy(state.text, inp.text)
	inp.undoStack.Push(state)
}

// Render writes the input line into the buffer with optional cursor marker.
func (inp *Input) Render(out *tui.RenderBuffer, w int) {
	if len(inp.text) == 0 && inp.placeholder != "" && inp.focused {
		line := "\x1b[2m" + inp.placeholder + "\x1b[0m"
		if inp.focused {
			line = tui.CursorMarker + line
		}
		out.WriteLine(line)
		return
	}

	if len(inp.text) == 0 && !inp.focused {
		out.WriteLine("")
		return
	}

	displayText := string(inp.text)

	if !inp.focused {
		out.WriteLine(width.TruncateToWidth(displayText, w))
		return
	}

	// Calculate scroll offset for horizontal scrolling
	inp.updateScrollOffset(w)

	var b strings.Builder
	visibleStart := inp.scrollOff
	visibleEnd := visibleStart + w - 1 // leave room for cursor
	if visibleEnd > len(inp.text) {
		visibleEnd = len(inp.text)
	}

	for i := visibleStart; i < visibleEnd; i++ {
		if i == inp.cursor {
			b.WriteString(tui.CursorMarker)
		}
		b.WriteRune(inp.text[i])
	}
	if inp.cursor >= visibleEnd {
		b.WriteString(tui.CursorMarker)
	}

	out.WriteLine(b.String())
	inp.dirty = false
}

func (inp *Input) updateScrollOffset(w int) {
	if w <= 0 {
		return
	}
	// Ensure cursor is visible
	if inp.cursor < inp.scrollOff {
		inp.scrollOff = inp.cursor
	}
	if inp.cursor >= inp.scrollOff+w {
		inp.scrollOff = inp.cursor - w + 1
	}
}
