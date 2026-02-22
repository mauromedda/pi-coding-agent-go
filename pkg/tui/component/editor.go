// ABOUTME: Multi-line text editor component with word-wrap, cursor tracking, undo/redo
// ABOUTME: Supports Emacs-style keybindings, kill ring, and line-based editing operations
// ABOUTME: Supports Ctrl+V for image paste from clipboard

package component

import (
	"strings"
	"sync"

	"github.com/mauromedda/pi-coding-agent-go/pkg/tui"
	"github.com/mauromedda/pi-coding-agent-go/pkg/tui/image"
	"github.com/mauromedda/pi-coding-agent-go/pkg/tui/internal/killring"
	"github.com/mauromedda/pi-coding-agent-go/pkg/tui/internal/undo"
	"github.com/mauromedda/pi-coding-agent-go/pkg/tui/key"
	"github.com/mauromedda/pi-coding-agent-go/pkg/tui/width"
)

const editorUndoDepth = 200

// editorState captures the full editor state for undo/redo.
type editorState struct {
	lines [][]rune
	row   int
	col   int
}

// Editor is a multi-line text editor with word-wrap display and cursor tracking.
// Thread-safe: mu protects all mutable state for concurrent access from
// the StdinBuffer goroutine (HandleKey) and the render goroutine (Render).
type Editor struct {
	mu        sync.Mutex
	lines     [][]rune
	row       int
	col       int
	focused   bool
	dirty     bool
	ring      *killring.KillRing
	undoStack *undo.Stack[editorState]

	// Prompt prefix prepended to line 0 render (e.g. "❯ ")
	prompt      string
	promptWidth int
	// Placeholder text shown dim when empty and focused
	placeholder string
}

// NewEditor creates a new empty Editor component.
func NewEditor() *Editor {
	return &Editor{
		lines:     [][]rune{{}},
		ring:      killring.New(),
		undoStack: undo.New[editorState](editorUndoDepth),
		dirty:     true,
	}
}

// Text returns the full editor content as a string with newline separators.
func (ed *Editor) Text() string {
	ed.mu.Lock()
	defer ed.mu.Unlock()

	parts := make([]string, len(ed.lines))
	for i, line := range ed.lines {
		parts[i] = string(line)
	}
	return strings.Join(parts, "\n")
}

// SetText replaces the editor content and resets the cursor.
func (ed *Editor) SetText(s string) {
	ed.mu.Lock()
	defer ed.mu.Unlock()

	raw := splitLines(s)
	ed.lines = make([][]rune, len(raw))
	for i, l := range raw {
		ed.lines[i] = []rune(l)
	}
	ed.row = len(ed.lines) - 1
	ed.col = len(ed.lines[ed.row])
	ed.dirty = true
}

// CursorPos returns the cursor position as (row, col).
func (ed *Editor) CursorPos() (int, int) {
	ed.mu.Lock()
	defer ed.mu.Unlock()

	return ed.row, ed.col
}

// SetFocused sets the focus state.
func (ed *Editor) SetFocused(focused bool) {
	ed.mu.Lock()
	defer ed.mu.Unlock()

	ed.focused = focused
	ed.dirty = true
}

// IsFocused returns the focus state.
func (ed *Editor) IsFocused() bool {
	ed.mu.Lock()
	defer ed.mu.Unlock()

	return ed.focused
}

// Invalidate marks the component for re-render.
func (ed *Editor) Invalidate() {
	ed.mu.Lock()
	defer ed.mu.Unlock()

	ed.dirty = true
}

// SetPrompt sets a prefix string that is prepended to line 0 during render.
// Subsequent lines are indented by the prompt's visible width.
func (ed *Editor) SetPrompt(p string) {
	ed.mu.Lock()
	defer ed.mu.Unlock()

	ed.prompt = p
	ed.promptWidth = width.VisibleWidth(p)
	ed.dirty = true
}

// SetPlaceholder sets dim hint text shown when the editor is empty and focused.
func (ed *Editor) SetPlaceholder(p string) {
	ed.mu.Lock()
	defer ed.mu.Unlock()

	ed.placeholder = p
	ed.dirty = true
}

// isEmpty returns true if the editor contains no text.
// Must be called with ed.mu held.
func (ed *Editor) isEmpty() bool {
	return len(ed.lines) == 1 && len(ed.lines[0]) == 0
}

// HandleInput processes raw terminal input data.
func (ed *Editor) HandleInput(data string) {
	ed.mu.Lock()
	defer ed.mu.Unlock()

	k := key.ParseKey(data)
	ed.dispatchKey(k, data)
}

// HandleKey processes an already-parsed key event. Thread-safe.
func (ed *Editor) HandleKey(k key.Key) {
	ed.mu.Lock()
	defer ed.mu.Unlock()

	ed.dispatchKey(k, "")
}

// dispatchKey routes a key to the appropriate editing operation.
// Must be called with ed.mu held.
func (ed *Editor) dispatchKey(k key.Key, rawData string) {
	// Handle Ctrl+key combinations via the Key struct
	if k.Ctrl && k.Type == key.KeyRune {
		switch k.Rune {
		case 'a':
			ed.moveCursorHome()
			return
		case 'e':
			ed.moveCursorEnd()
			return
		case 'k':
			ed.killToEnd()
			return
		case 'y':
			ed.yank()
			return
		case 'z':
			ed.doUndo()
			return
		case 'v': // Ctrl+V = paste image
			ed.pasteImage()
			return
		}
	}

	switch k.Type {
	case key.KeyRune:
		ed.insertRune(k.Rune)
	case key.KeyEnter:
		ed.insertNewline()
	case key.KeyBackspace:
		ed.backspace()
	case key.KeyDelete:
		ed.delete()
	case key.KeyLeft:
		ed.moveCursorLeft()
	case key.KeyRight:
		ed.moveCursorRight()
	case key.KeyUp:
		ed.moveCursorUp()
	case key.KeyDown:
		ed.moveCursorDown()
	case key.KeyHome:
		ed.moveCursorHome()
	case key.KeyEnd:
		ed.moveCursorEnd()
	default:
		if rawData != "" {
			ed.handleControlByte(rawData)
		}
	}
}

func (ed *Editor) handleControlByte(data string) {
	if len(data) != 1 {
		return
	}
	switch data[0] {
	case 0x01: // Ctrl+A = home
		ed.moveCursorHome()
	case 0x05: // Ctrl+E = end
		ed.moveCursorEnd()
	case 0x0b: // Ctrl+K = kill to end of line
		ed.killToEnd()
	case 0x19: // Ctrl+Y = yank
		ed.yank()
	case 0x1a: // Ctrl+Z = undo
		ed.doUndo()
	}
}

func (ed *Editor) insertRune(r rune) {
	ed.saveUndo()
	line := ed.lines[ed.row]
	newLine := make([]rune, len(line)+1)
	copy(newLine, line[:ed.col])
	newLine[ed.col] = r
	copy(newLine[ed.col+1:], line[ed.col:])
	ed.lines[ed.row] = newLine
	ed.col++
	ed.dirty = true
}

func (ed *Editor) insertNewline() {
	ed.saveUndo()
	line := ed.lines[ed.row]
	before := make([]rune, ed.col)
	copy(before, line[:ed.col])
	after := make([]rune, len(line)-ed.col)
	copy(after, line[ed.col:])

	ed.lines[ed.row] = before

	// Insert new line after current
	newLines := make([][]rune, len(ed.lines)+1)
	copy(newLines, ed.lines[:ed.row+1])
	newLines[ed.row+1] = after
	copy(newLines[ed.row+2:], ed.lines[ed.row+1:])
	ed.lines = newLines

	ed.row++
	ed.col = 0
	ed.dirty = true
}

func (ed *Editor) backspace() {
	if ed.col > 0 {
		ed.saveUndo()
		line := ed.lines[ed.row]
		ed.lines[ed.row] = append(line[:ed.col-1], line[ed.col:]...)
		ed.col--
		ed.dirty = true
		return
	}
	// At start of line; join with previous
	if ed.row == 0 {
		return
	}
	ed.saveUndo()
	prevLen := len(ed.lines[ed.row-1])
	ed.lines[ed.row-1] = append(ed.lines[ed.row-1], ed.lines[ed.row]...)
	ed.lines = append(ed.lines[:ed.row], ed.lines[ed.row+1:]...)
	ed.row--
	ed.col = prevLen
	ed.dirty = true
}

func (ed *Editor) delete() {
	line := ed.lines[ed.row]
	if ed.col < len(line) {
		ed.saveUndo()
		ed.lines[ed.row] = append(line[:ed.col], line[ed.col+1:]...)
		ed.dirty = true
		return
	}
	// At end of line; join with next
	if ed.row >= len(ed.lines)-1 {
		return
	}
	ed.saveUndo()
	ed.lines[ed.row] = append(ed.lines[ed.row], ed.lines[ed.row+1]...)
	ed.lines = append(ed.lines[:ed.row+1], ed.lines[ed.row+2:]...)
	ed.dirty = true
}

func (ed *Editor) moveCursorLeft() {
	if ed.col > 0 {
		ed.col--
	} else if ed.row > 0 {
		ed.row--
		ed.col = len(ed.lines[ed.row])
	}
	ed.dirty = true
}

func (ed *Editor) moveCursorRight() {
	if ed.col < len(ed.lines[ed.row]) {
		ed.col++
	} else if ed.row < len(ed.lines)-1 {
		ed.row++
		ed.col = 0
	}
	ed.dirty = true
}

func (ed *Editor) moveCursorUp() {
	if ed.row > 0 {
		ed.row--
		if ed.col > len(ed.lines[ed.row]) {
			ed.col = len(ed.lines[ed.row])
		}
		ed.dirty = true
	}
}

func (ed *Editor) moveCursorDown() {
	if ed.row < len(ed.lines)-1 {
		ed.row++
		if ed.col > len(ed.lines[ed.row]) {
			ed.col = len(ed.lines[ed.row])
		}
		ed.dirty = true
	}
}

func (ed *Editor) moveCursorHome() {
	ed.col = 0
	ed.dirty = true
}

func (ed *Editor) moveCursorEnd() {
	ed.col = len(ed.lines[ed.row])
	ed.dirty = true
}

func (ed *Editor) killToEnd() {
	line := ed.lines[ed.row]
	if ed.col >= len(line) {
		return
	}
	ed.saveUndo()
	killed := string(line[ed.col:])
	ed.ring.Push(killed)
	ed.lines[ed.row] = line[:ed.col]
	ed.dirty = true
}

func (ed *Editor) yank() {
	yanked := ed.ring.Yank()
	if yanked == "" {
		return
	}
	ed.saveUndo()
	runes := []rune(yanked)
	line := ed.lines[ed.row]
	newLine := make([]rune, 0, len(line)+len(runes))
	newLine = append(newLine, line[:ed.col]...)
	newLine = append(newLine, runes...)
	newLine = append(newLine, line[ed.col:]...)
	ed.lines[ed.row] = newLine
	ed.col += len(runes)
	ed.dirty = true
}

func (ed *Editor) doUndo() {
	state, ok := ed.undoStack.Undo()
	if !ok {
		return
	}
	ed.lines = state.lines
	ed.row = state.row
	ed.col = state.col
	ed.dirty = true
}

// pasteImage pastes an image from the clipboard as a reference.
// Uses the image package to detect the protocol and get clipboard data.
func (ed *Editor) pasteImage() {
	// Try to paste image
	if img, err := image.ClipboardImage(); err == nil && len(img) > 0 {
		// Insert image placeholder
		ed.insertRune('[')
		ed.insertText("Image")
		ed.insertText("]")
		ed.insertNewline()
		// Add placeholder with file size
		ed.insertText(image.ImagePlaceholder(img))
		ed.insertNewline()
	} else {
		// No image in clipboard - just insert a newline
		ed.insertNewline()
	}
}

func (ed *Editor) insertText(text string) {
	ed.saveUndo()
	line := ed.lines[ed.row]
	runes := []rune(text)
	newLine := make([]rune, len(line)+len(runes))
	copy(newLine, line[:ed.col])
	copy(newLine[ed.col:], runes)
	copy(newLine[ed.col+len(runes):], line[ed.col:])
	ed.lines[ed.row] = newLine
	ed.col += len(runes)
	ed.dirty = true
}

func (ed *Editor) saveUndo() {
	lines := make([][]rune, len(ed.lines))
	for i, l := range ed.lines {
		lines[i] = make([]rune, len(l))
		copy(lines[i], l)
	}
	ed.undoStack.Push(editorState{
		lines: lines,
		row:   ed.row,
		col:   ed.col,
	})
}

// Render writes the editor content into the buffer with word-wrap.
func (ed *Editor) Render(out *tui.RenderBuffer, w int) {
	ed.mu.Lock()
	defer ed.mu.Unlock()

	if w <= 0 {
		return
	}

	// Effective width for text content (reduced by prompt prefix)
	ew := max(w-ed.promptWidth, 1)

	// Placeholder: shown when empty, focused, and placeholder is set
	if ed.focused && ed.isEmpty() && ed.placeholder != "" {
		out.WriteLine(ed.prompt + tui.CursorMarker + "\x1b[2m" + ed.placeholder + "\x1b[0m")
		return
	}

	indent := strings.Repeat(" ", ed.promptWidth)

	for i, line := range ed.lines {
		lineStr := string(line)
		wrapped := width.WrapTextWithAnsi(lineStr, ew)

		// Prepend prompt (line 0) or indent (subsequent lines)
		prefix := indent
		if i == 0 {
			prefix = ed.prompt
		}

		if !ed.focused {
			for wi, wl := range wrapped {
				if wi == 0 {
					out.WriteLine(prefix + wl)
				} else {
					out.WriteLine(indent + wl)
				}
			}
			continue
		}

		// If cursor is on this line, insert the cursor marker
		if i == ed.row {
			ed.renderLineWithCursor(out, wrapped, line, ew, prefix, indent)
		} else {
			for wi, wl := range wrapped {
				if wi == 0 {
					out.WriteLine(prefix + wl)
				} else {
					out.WriteLine(indent + wl)
				}
			}
		}
	}
}

func (ed *Editor) renderLineWithCursor(out *tui.RenderBuffer, wrapped []string, line []rune, ew int, prefix, indent string) {
	// Find which wrapped line the cursor falls on
	cursorOffset := ed.col
	wrapRow := 0
	for wrapRow < len(wrapped)-1 && cursorOffset >= ew {
		cursorOffset -= ew
		wrapRow++
	}

	for wi, wl := range wrapped {
		// Determine line prefix
		lp := indent
		if wi == 0 {
			lp = prefix
		}

		if wi == wrapRow {
			// Insert cursor marker at the correct position within this wrapped line
			runes := []rune(width.StripANSI(wl))
			var b strings.Builder
			if cursorOffset > len(runes) {
				cursorOffset = len(runes)
			}
			b.WriteString(lp)
			b.WriteString(string(runes[:cursorOffset]))
			b.WriteString(tui.CursorMarker)
			if cursorOffset < len(runes) {
				b.WriteString(string(runes[cursorOffset:]))
			}
			out.WriteLine(b.String())
		} else {
			out.WriteLine(lp + wl)
		}
	}
}
