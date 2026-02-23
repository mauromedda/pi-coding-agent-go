// ABOUTME: EditorModel is a Bubble Tea multi-line rune editor with kill ring and undo
// ABOUTME: Port of pkg/tui/component/editor.go; value semantics, no mutex needed

package btea

import (
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/mauromedda/pi-coding-agent-go/pkg/tui/image"
	"github.com/mauromedda/pi-coding-agent-go/pkg/tui/width"
)

// --- Inline kill ring (cannot import pkg/tui/internal/killring) ---

const killRingSize = 32

// killRing is a minimal Emacs-style ring buffer for killed (cut) text.
type killRing struct {
	entries []string
	pos     int
	size    int
}

func newKillRing() *killRing {
	return &killRing{
		entries: make([]string, 0, killRingSize),
		size:    killRingSize,
	}
}

func (kr *killRing) push(text string) {
	if len(kr.entries) < kr.size {
		kr.entries = append(kr.entries, text)
	} else {
		kr.entries[kr.pos] = text
	}
	kr.pos = (kr.pos + 1) % kr.size
}

func (kr *killRing) yank() string {
	if len(kr.entries) == 0 {
		return ""
	}
	idx := (kr.pos - 1 + len(kr.entries)) % len(kr.entries)
	return kr.entries[idx]
}

// --- Inline undo stack (cannot import pkg/tui/internal/undo) ---

// undoStack is a generic undo stack for editor state snapshots.
type undoStack[S any] struct {
	items   []S
	maxSize int
}

func newUndoStack[S any](maxSize int) *undoStack[S] {
	return &undoStack[S]{
		items:   make([]S, 0, maxSize),
		maxSize: maxSize,
	}
}

func (s *undoStack[S]) push(state S) {
	if len(s.items) >= s.maxSize {
		s.items = s.items[1:]
	}
	s.items = append(s.items, state)
}

func (s *undoStack[S]) undo() (S, bool) {
	if len(s.items) == 0 {
		var zero S
		return zero, false
	}
	last := s.items[len(s.items)-1]
	s.items = s.items[:len(s.items)-1]
	return last, true
}

const editorUndoDepth = 200

// CursorMarker is the visible block cursor character.
const CursorMarker = "█"

// editorState captures the full editor state for undo/redo.
type editorState struct {
	lines [][]rune
	row   int
	col   int
}

// EditorModel is a multi-line text editor with kill ring, undo stack,
// word-wrap rendering, and cursor tracking.
// Implements tea.Model. The kill ring and undo stack are pointer types
// shared across value copies, which is the correct Bubble Tea pattern
// (same as bubbles/textarea). Only one copy is in use at a time.
type EditorModel struct {
	lines       [][]rune
	row, col    int
	focused     bool
	ring        *killRing
	undoStack   *undoStack[editorState]
	prompt      string
	promptWidth int
	placeholder string
	width       int
	ghostText   string // dimmed completion shown after cursor
}

// NewEditorModel creates a new empty editor.
func NewEditorModel() EditorModel {
	return EditorModel{
		lines:     [][]rune{{}},
		ring:      newKillRing(),
		undoStack: newUndoStack[editorState](editorUndoDepth),
	}
}

// Init returns nil; no commands needed at startup.
func (m EditorModel) Init() tea.Cmd {
	return nil
}

// Update handles key and window-size messages.
func (m EditorModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		m.dispatchKey(msg)
	case tea.WindowSizeMsg:
		m.width = msg.Width
	}
	return m, nil
}

// View renders the editor content with word-wrap and cursor.
func (m EditorModel) View() string {
	if m.width <= 0 {
		return ""
	}

	s := Styles()
	ew := max(m.width-m.promptWidth, 1)

	// Placeholder: shown when empty, focused, and placeholder is set
	if m.focused && m.isEmpty() && m.placeholder != "" {
		return m.prompt + CursorMarker + s.Dim.Render(m.placeholder)
	}

	indent := strings.Repeat(" ", m.promptWidth)
	var b strings.Builder

	for i, line := range m.lines {
		lineStr := string(line)
		wrapped := width.WrapTextWithAnsi(lineStr, ew)

		prefix := indent
		if i == 0 {
			prefix = m.prompt
		}

		if m.focused && i == m.row {
			m.renderLineWithCursor(&b, wrapped, line, ew, prefix, indent, i > 0)
		} else {
			for wi, wl := range wrapped {
				if i > 0 || wi > 0 {
					b.WriteByte('\n')
				}
				if wi == 0 {
					b.WriteString(prefix + wl)
				} else {
					b.WriteString(indent + wl)
				}
			}
		}
	}

	return b.String()
}

// --- Public methods (value receivers, return new model) ---

// Text returns the full editor content as a string with newline separators.
func (m EditorModel) Text() string {
	parts := make([]string, len(m.lines))
	for i, line := range m.lines {
		parts[i] = string(line)
	}
	return strings.Join(parts, "\n")
}

// SetText replaces the editor content and places cursor at end.
func (m EditorModel) SetText(s string) EditorModel {
	raw := splitLines(s)
	m.lines = make([][]rune, len(raw))
	for i, l := range raw {
		m.lines[i] = []rune(l)
	}
	m.row = len(m.lines) - 1
	m.col = len(m.lines[m.row])
	return m
}

// CursorPos returns the cursor position as (row, col).
func (m EditorModel) CursorPos() (int, int) {
	return m.row, m.col
}

// SetFocused sets the focus state. Returns a new model.
func (m EditorModel) SetFocused(focused bool) EditorModel {
	m.focused = focused
	return m
}

// SetPrompt sets the prompt prefix for line 0. Returns a new model.
func (m EditorModel) SetPrompt(p string) EditorModel {
	m.prompt = p
	m.promptWidth = width.VisibleWidth(p)
	return m
}

// SetPlaceholder sets dim hint text shown when empty and focused. Returns a new model.
func (m EditorModel) SetPlaceholder(p string) EditorModel {
	m.placeholder = p
	return m
}

// IsEmpty returns true if the editor contains no text.
func (m EditorModel) IsEmpty() bool {
	return len(m.lines) == 1 && len(m.lines[0]) == 0
}

// SetGhostText sets dimmed completion text shown after the cursor.
func (m EditorModel) SetGhostText(g string) EditorModel {
	m.ghostText = g
	return m
}

// GhostText returns the current ghost text.
func (m EditorModel) GhostText() string {
	return m.ghostText
}

// CursorRow returns the zero-based row index of the cursor.
func (m EditorModel) CursorRow() int {
	return m.row
}

// LineCount returns the number of lines in the editor buffer.
func (m EditorModel) LineCount() int {
	return len(m.lines)
}

// --- Key dispatch ---

func (m *EditorModel) dispatchKey(msg tea.KeyMsg) {
	switch msg.Type {
	case tea.KeyRunes:
		if len(msg.Runes) > 0 {
			m.insertRune(msg.Runes[0])
		}
	case tea.KeySpace:
		m.insertRune(' ')
	case tea.KeyTab:
		if m.ghostText != "" {
			m.acceptGhostText()
		} else {
			m.insertRune('\t')
		}
	case tea.KeyEnter:
		m.insertNewline()
	case tea.KeyBackspace:
		m.backspace()
	case tea.KeyDelete:
		m.delete()
	case tea.KeyLeft:
		m.moveCursorLeft()
	case tea.KeyRight:
		m.moveCursorRight()
	case tea.KeyUp:
		m.moveCursorUp()
	case tea.KeyDown:
		m.moveCursorDown()
	case tea.KeyHome:
		m.moveCursorHome()
	case tea.KeyEnd:
		m.moveCursorEnd()
	case tea.KeyCtrlA:
		m.moveCursorHome()
	case tea.KeyCtrlE:
		m.moveCursorEnd()
	case tea.KeyCtrlK:
		m.killToEnd()
	case tea.KeyCtrlY:
		m.yank()
	case tea.KeyCtrlZ:
		m.doUndo()
	case tea.KeyCtrlV:
		m.pasteImage()
	}
}

// acceptGhostText inserts the ghost text at the cursor position and clears it.
func (m *EditorModel) acceptGhostText() {
	if m.ghostText == "" {
		return
	}
	m.insertText(m.ghostText)
	m.ghostText = ""
}

// --- Editing operations ---

func (m *EditorModel) insertRune(r rune) {
	m.saveUndo()
	line := m.lines[m.row]
	newLine := make([]rune, len(line)+1)
	copy(newLine, line[:m.col])
	newLine[m.col] = r
	copy(newLine[m.col+1:], line[m.col:])
	m.lines[m.row] = newLine
	m.col++
}

func (m *EditorModel) insertNewline() {
	m.saveUndo()
	line := m.lines[m.row]
	before := make([]rune, m.col)
	copy(before, line[:m.col])
	after := make([]rune, len(line)-m.col)
	copy(after, line[m.col:])

	m.lines[m.row] = before

	newLines := make([][]rune, len(m.lines)+1)
	copy(newLines, m.lines[:m.row+1])
	newLines[m.row+1] = after
	copy(newLines[m.row+2:], m.lines[m.row+1:])
	m.lines = newLines

	m.row++
	m.col = 0
}

func (m *EditorModel) backspace() {
	if m.col > 0 {
		m.saveUndo()
		line := m.lines[m.row]
		m.lines[m.row] = append(line[:m.col-1], line[m.col:]...)
		m.col--
		return
	}
	if m.row == 0 {
		return
	}
	m.saveUndo()
	prevLen := len(m.lines[m.row-1])
	m.lines[m.row-1] = append(m.lines[m.row-1], m.lines[m.row]...)
	m.lines = append(m.lines[:m.row], m.lines[m.row+1:]...)
	m.row--
	m.col = prevLen
}

func (m *EditorModel) delete() {
	line := m.lines[m.row]
	if m.col < len(line) {
		m.saveUndo()
		m.lines[m.row] = append(line[:m.col], line[m.col+1:]...)
		return
	}
	if m.row >= len(m.lines)-1 {
		return
	}
	m.saveUndo()
	m.lines[m.row] = append(m.lines[m.row], m.lines[m.row+1]...)
	m.lines = append(m.lines[:m.row+1], m.lines[m.row+2:]...)
}

func (m *EditorModel) moveCursorLeft() {
	if m.col > 0 {
		m.col--
	} else if m.row > 0 {
		m.row--
		m.col = len(m.lines[m.row])
	}
}

func (m *EditorModel) moveCursorRight() {
	if m.col < len(m.lines[m.row]) {
		m.col++
	} else if m.row < len(m.lines)-1 {
		m.row++
		m.col = 0
	}
}

func (m *EditorModel) moveCursorUp() {
	if m.row > 0 {
		m.row--
		if m.col > len(m.lines[m.row]) {
			m.col = len(m.lines[m.row])
		}
	}
}

func (m *EditorModel) moveCursorDown() {
	if m.row < len(m.lines)-1 {
		m.row++
		if m.col > len(m.lines[m.row]) {
			m.col = len(m.lines[m.row])
		}
	}
}

func (m *EditorModel) moveCursorHome() {
	m.col = 0
}

func (m *EditorModel) moveCursorEnd() {
	m.col = len(m.lines[m.row])
}

func (m *EditorModel) killToEnd() {
	line := m.lines[m.row]
	if m.col >= len(line) {
		return
	}
	m.saveUndo()
	killed := string(line[m.col:])
	m.ring.push(killed)
	m.lines[m.row] = line[:m.col]
}

func (m *EditorModel) yank() {
	yanked := m.ring.yank()
	if yanked == "" {
		return
	}
	m.saveUndo()
	runes := []rune(yanked)
	line := m.lines[m.row]
	newLine := make([]rune, 0, len(line)+len(runes))
	newLine = append(newLine, line[:m.col]...)
	newLine = append(newLine, runes...)
	newLine = append(newLine, line[m.col:]...)
	m.lines[m.row] = newLine
	m.col += len(runes)
}

func (m *EditorModel) doUndo() {
	state, ok := m.undoStack.undo()
	if !ok {
		return
	}
	// Deep copy to prevent mutating the snapshot in the stack
	m.lines = make([][]rune, len(state.lines))
	for i, l := range state.lines {
		m.lines[i] = make([]rune, len(l))
		copy(m.lines[i], l)
	}
	m.row = state.row
	m.col = state.col
}

func (m *EditorModel) pasteImage() {
	img, err := image.ClipboardImage()
	if err == nil && len(img) > 0 {
		m.insertRune('[')
		m.insertText("Image")
		m.insertText("]")
		m.insertNewline()
		m.insertText(image.ImagePlaceholder(img))
		m.insertNewline()
	} else {
		m.insertNewline()
	}
}

func (m *EditorModel) insertText(text string) {
	m.saveUndo()
	line := m.lines[m.row]
	runes := []rune(text)
	newLine := make([]rune, len(line)+len(runes))
	copy(newLine, line[:m.col])
	copy(newLine[m.col:], runes)
	copy(newLine[m.col+len(runes):], line[m.col:])
	m.lines[m.row] = newLine
	m.col += len(runes)
}

func (m *EditorModel) saveUndo() {
	lines := make([][]rune, len(m.lines))
	for i, l := range m.lines {
		lines[i] = make([]rune, len(l))
		copy(lines[i], l)
	}
	m.undoStack.push(editorState{
		lines: lines,
		row:   m.row,
		col:   m.col,
	})
}

func (m *EditorModel) isEmpty() bool {
	return len(m.lines) == 1 && len(m.lines[0]) == 0
}

// --- View helpers ---

func (m *EditorModel) renderLineWithCursor(b *strings.Builder, wrapped []string, line []rune, ew int, prefix, indent string, needNewline bool) {
	cursorOffset := m.col
	wrapRow := 0
	for wrapRow < len(wrapped)-1 && cursorOffset >= ew {
		cursorOffset -= ew
		wrapRow++
	}

	for wi, wl := range wrapped {
		if needNewline || wi > 0 {
			b.WriteByte('\n')
		}

		lp := indent
		if wi == 0 {
			lp = prefix
		}

		if wi == wrapRow {
			runes := []rune(width.StripANSI(wl))
			if cursorOffset > len(runes) {
				cursorOffset = len(runes)
			}
			b.WriteString(lp)
			b.WriteString(string(runes[:cursorOffset]))
			b.WriteString(CursorMarker)
			if cursorOffset < len(runes) {
				b.WriteString(string(runes[cursorOffset:]))
			}
			// Render ghost text after cursor if at end of line
			if m.ghostText != "" && cursorOffset >= len(runes) {
				s := Styles()
				b.WriteString(s.Dim.Render(m.ghostText))
			}
		} else {
			b.WriteString(lp + wl)
		}
	}
}

// splitLines splits a string into lines, preserving the invariant that
// an empty string produces a single empty line.
func splitLines(s string) []string {
	if s == "" {
		return []string{""}
	}
	return strings.Split(s, "\n")
}
