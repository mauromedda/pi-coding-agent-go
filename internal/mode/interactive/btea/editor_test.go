// ABOUTME: Tests for EditorModel Bubble Tea leaf component
// ABOUTME: Verifies rune editing, cursor nav, kill/yank, undo, newlines, View rendering

package btea

import (
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
)

// Compile-time check: EditorModel must satisfy tea.Model.
var _ tea.Model = EditorModel{}

func TestEditorModel_Init(t *testing.T) {
	m := NewEditorModel()
	if cmd := m.Init(); cmd != nil {
		t.Errorf("Init() returned non-nil cmd")
	}
}

func TestEditorModel_NewStartsWithOneEmptyLine(t *testing.T) {
	m := NewEditorModel()
	if len(m.lines) != 1 {
		t.Errorf("lines count = %d; want 1", len(m.lines))
	}
	if len(m.lines[0]) != 0 {
		t.Errorf("first line len = %d; want 0", len(m.lines[0]))
	}
}

func TestEditorModel_TextReturnsEmpty(t *testing.T) {
	m := NewEditorModel()
	if got := m.Text(); got != "" {
		t.Errorf("Text() = %q; want empty", got)
	}
}

func TestEditorModel_IsEmptyOnNew(t *testing.T) {
	m := NewEditorModel()
	if !m.IsEmpty() {
		t.Errorf("IsEmpty() = false; want true")
	}
}

func TestEditorModel_SetText(t *testing.T) {
	m := NewEditorModel()
	m = m.SetText("hello\nworld")
	if got := m.Text(); got != "hello\nworld" {
		t.Errorf("Text() = %q; want %q", got, "hello\nworld")
	}
	row, col := m.CursorPos()
	if row != 1 || col != 5 {
		t.Errorf("CursorPos() = (%d, %d); want (1, 5)", row, col)
	}
	if m.IsEmpty() {
		t.Errorf("IsEmpty() = true after SetText; want false")
	}
}

func TestEditorModel_InsertRuneViaKeyMsg(t *testing.T) {
	m := NewEditorModel()
	// Type 'a', 'b', 'c'
	for _, r := range "abc" {
		updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{r}})
		m = updated.(EditorModel)
	}
	if got := m.Text(); got != "abc" {
		t.Errorf("Text() = %q; want %q", got, "abc")
	}
	row, col := m.CursorPos()
	if row != 0 || col != 3 {
		t.Errorf("CursorPos() = (%d, %d); want (0, 3)", row, col)
	}
}

func TestEditorModel_SpaceKeyInserts(t *testing.T) {
	m := NewEditorModel()
	// Type "hello world" — space is dispatched as tea.KeySpace
	for _, r := range "hello" {
		updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{r}})
		m = updated.(EditorModel)
	}
	updated, _ := m.Update(tea.KeyMsg{Type: tea.KeySpace, Runes: []rune{' '}})
	m = updated.(EditorModel)
	for _, r := range "world" {
		updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{r}})
		m = updated.(EditorModel)
	}
	if got := m.Text(); got != "hello world" {
		t.Errorf("Text() = %q; want %q", got, "hello world")
	}
}

func TestEditorModel_Backspace(t *testing.T) {
	m := NewEditorModel()
	m = m.SetText("abc")
	updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyBackspace})
	m = updated.(EditorModel)
	if got := m.Text(); got != "ab" {
		t.Errorf("after backspace: Text() = %q; want %q", got, "ab")
	}
}

func TestEditorModel_BackspaceAtStartJoinsLines(t *testing.T) {
	m := NewEditorModel()
	m = m.SetText("hello\nworld")
	// Cursor is at end of "world" (row=1, col=5). Move to start of line 1.
	m.row = 1
	m.col = 0
	updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyBackspace})
	m = updated.(EditorModel)
	if got := m.Text(); got != "helloworld" {
		t.Errorf("after backspace-join: Text() = %q; want %q", got, "helloworld")
	}
	row, col := m.CursorPos()
	if row != 0 || col != 5 {
		t.Errorf("CursorPos() = (%d, %d); want (0, 5)", row, col)
	}
}

func TestEditorModel_EnterCreatesNewLine(t *testing.T) {
	m := NewEditorModel()
	m = m.SetText("hello")
	// Cursor at end of "hello"
	updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	m = updated.(EditorModel)
	if got := m.Text(); got != "hello\n" {
		t.Errorf("after enter: Text() = %q; want %q", got, "hello\n")
	}
	row, col := m.CursorPos()
	if row != 1 || col != 0 {
		t.Errorf("CursorPos() = (%d, %d); want (1, 0)", row, col)
	}
}

func TestEditorModel_EnterSplitsLine(t *testing.T) {
	m := NewEditorModel()
	m = m.SetText("helloworld")
	m.col = 5 // Between "hello" and "world"
	updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	m = updated.(EditorModel)
	if got := m.Text(); got != "hello\nworld" {
		t.Errorf("after enter-split: Text() = %q; want %q", got, "hello\nworld")
	}
}

func TestEditorModel_Delete(t *testing.T) {
	m := NewEditorModel()
	m = m.SetText("abc")
	m.col = 1 // After 'a'
	updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyDelete})
	m = updated.(EditorModel)
	if got := m.Text(); got != "ac" {
		t.Errorf("after delete: Text() = %q; want %q", got, "ac")
	}
}

func TestEditorModel_DeleteJoinsLines(t *testing.T) {
	m := NewEditorModel()
	m = m.SetText("hello\nworld")
	m.row = 0
	m.col = 5 // End of "hello"
	updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyDelete})
	m = updated.(EditorModel)
	if got := m.Text(); got != "helloworld" {
		t.Errorf("after delete-join: Text() = %q; want %q", got, "helloworld")
	}
}

func TestEditorModel_KillToEnd(t *testing.T) {
	m := NewEditorModel()
	m = m.SetText("hello world")
	m.col = 5 // After "hello"
	// ctrl+k kills to end
	updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'k'}, Alt: false})
	// That sends 'k' as a rune. We need ctrl+k.
	// Bubble Tea represents ctrl+k as tea.KeyCtrlK
	m2 := NewEditorModel()
	m2 = m2.SetText("hello world")
	m2.col = 5
	updated, _ = m2.Update(tea.KeyMsg{Type: tea.KeyCtrlK})
	m2 = updated.(EditorModel)
	if got := m2.Text(); got != "hello" {
		t.Errorf("after ctrl+k: Text() = %q; want %q", got, "hello")
	}
}

func TestEditorModel_YankRestoresKilled(t *testing.T) {
	m := NewEditorModel()
	m = m.SetText("hello world")
	m.col = 5
	// Kill to end
	updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyCtrlK})
	m = updated.(EditorModel)
	if got := m.Text(); got != "hello" {
		t.Fatalf("after ctrl+k: Text() = %q; want %q", got, "hello")
	}
	// Yank
	updated, _ = m.Update(tea.KeyMsg{Type: tea.KeyCtrlY})
	m = updated.(EditorModel)
	if got := m.Text(); got != "hello world" {
		t.Errorf("after ctrl+y: Text() = %q; want %q", got, "hello world")
	}
}

func TestEditorModel_Undo(t *testing.T) {
	m := NewEditorModel()
	// Type 'a'
	updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'a'}})
	m = updated.(EditorModel)
	if got := m.Text(); got != "a" {
		t.Fatalf("after insert 'a': Text() = %q; want %q", got, "a")
	}
	// Undo
	updated, _ = m.Update(tea.KeyMsg{Type: tea.KeyCtrlZ})
	m = updated.(EditorModel)
	if got := m.Text(); got != "" {
		t.Errorf("after undo: Text() = %q; want empty", got)
	}
}

func TestEditorModel_CursorLeft(t *testing.T) {
	m := NewEditorModel()
	m = m.SetText("abc")
	// Cursor at col=3 (end)
	updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyLeft})
	m = updated.(EditorModel)
	_, col := m.CursorPos()
	if col != 2 {
		t.Errorf("after left: col = %d; want 2", col)
	}
}

func TestEditorModel_CursorRight(t *testing.T) {
	m := NewEditorModel()
	m = m.SetText("abc")
	m.col = 1
	updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyRight})
	m = updated.(EditorModel)
	_, col := m.CursorPos()
	if col != 2 {
		t.Errorf("after right: col = %d; want 2", col)
	}
}

func TestEditorModel_CursorUpDown(t *testing.T) {
	m := NewEditorModel()
	m = m.SetText("line1\nline2\nline3")
	// Cursor at (2, 5) end of "line3"
	if m.row != 2 {
		t.Fatalf("initial row = %d; want 2", m.row)
	}

	// Up
	updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyUp})
	m = updated.(EditorModel)
	if m.row != 1 {
		t.Errorf("after up: row = %d; want 1", m.row)
	}

	// Down
	updated, _ = m.Update(tea.KeyMsg{Type: tea.KeyDown})
	m = updated.(EditorModel)
	if m.row != 2 {
		t.Errorf("after down: row = %d; want 2", m.row)
	}
}

func TestEditorModel_CursorHome(t *testing.T) {
	m := NewEditorModel()
	m = m.SetText("hello")
	updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyHome})
	m = updated.(EditorModel)
	_, col := m.CursorPos()
	if col != 0 {
		t.Errorf("after home: col = %d; want 0", col)
	}
}

func TestEditorModel_CursorEnd(t *testing.T) {
	m := NewEditorModel()
	m = m.SetText("hello")
	m.col = 0
	updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyEnd})
	m = updated.(EditorModel)
	_, col := m.CursorPos()
	if col != 5 {
		t.Errorf("after end: col = %d; want 5", col)
	}
}

func TestEditorModel_CtrlA_Home(t *testing.T) {
	m := NewEditorModel()
	m = m.SetText("hello")
	updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyCtrlA})
	m = updated.(EditorModel)
	_, col := m.CursorPos()
	if col != 0 {
		t.Errorf("after ctrl+a: col = %d; want 0", col)
	}
}

func TestEditorModel_CtrlE_End(t *testing.T) {
	m := NewEditorModel()
	m = m.SetText("hello")
	m.col = 0
	updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyCtrlE})
	m = updated.(EditorModel)
	_, col := m.CursorPos()
	if col != 5 {
		t.Errorf("after ctrl+e: col = %d; want 5", col)
	}
}

func TestEditorModel_SetFocused(t *testing.T) {
	m := NewEditorModel()
	if m.focused {
		t.Errorf("initial focused = true; want false")
	}
	m = m.SetFocused(true)
	if !m.focused {
		t.Errorf("after SetFocused(true): focused = false")
	}
}

func TestEditorModel_SetPrompt(t *testing.T) {
	m := NewEditorModel()
	m = m.SetPrompt("> ")
	if m.prompt != "> " {
		t.Errorf("prompt = %q; want %q", m.prompt, "> ")
	}
}

func TestEditorModel_SetPlaceholder(t *testing.T) {
	m := NewEditorModel()
	m = m.SetPlaceholder("Type here...")
	if m.placeholder != "Type here..." {
		t.Errorf("placeholder = %q; want %q", m.placeholder, "Type here...")
	}
}

func TestEditorModel_ViewPromptAndPlaceholder(t *testing.T) {
	m := NewEditorModel()
	m = m.SetFocused(true)
	m = m.SetPrompt("> ")
	m = m.SetPlaceholder("Type here...")
	m.width = 80
	view := m.View()

	if !strings.Contains(view, "> ") {
		t.Errorf("View() missing prompt '> '")
	}
	if !strings.Contains(view, "Type here...") {
		t.Errorf("View() missing placeholder 'Type here...'")
	}
	if !strings.Contains(view, CursorMarker) {
		t.Errorf("View() missing cursor marker")
	}
}

func TestEditorModel_ViewWithContent(t *testing.T) {
	m := NewEditorModel()
	m = m.SetFocused(true)
	m = m.SetText("hello")
	m.width = 80
	view := m.View()

	if !strings.Contains(view, "hello") {
		t.Errorf("View() missing content 'hello'")
	}
}

func TestEditorModel_WindowSizeMsg(t *testing.T) {
	m := NewEditorModel()
	updated, cmd := m.Update(tea.WindowSizeMsg{Width: 100, Height: 30})
	if cmd != nil {
		t.Errorf("Update(WindowSizeMsg) returned non-nil cmd")
	}
	w := updated.(EditorModel)
	if w.width != 100 {
		t.Errorf("width = %d; want 100", w.width)
	}
}

func TestEditorModel_CursorLeftWrapsToEndOfPreviousLine(t *testing.T) {
	m := NewEditorModel()
	m = m.SetText("hello\nworld")
	m.row = 1
	m.col = 0
	updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyLeft})
	m = updated.(EditorModel)
	row, col := m.CursorPos()
	if row != 0 || col != 5 {
		t.Errorf("after left at start of line 1: CursorPos() = (%d, %d); want (0, 5)", row, col)
	}
}

func TestEditorModel_CursorRightWrapsToStartOfNextLine(t *testing.T) {
	m := NewEditorModel()
	m = m.SetText("hello\nworld")
	m.row = 0
	m.col = 5
	updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyRight})
	m = updated.(EditorModel)
	row, col := m.CursorPos()
	if row != 1 || col != 0 {
		t.Errorf("after right at end of line 0: CursorPos() = (%d, %d); want (1, 0)", row, col)
	}
}

func TestEditorModel_CursorUpClampsCol(t *testing.T) {
	m := NewEditorModel()
	m = m.SetText("hi\nhello")
	// Cursor at (1, 5) "hello"
	updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyUp})
	m = updated.(EditorModel)
	row, col := m.CursorPos()
	// "hi" is only 2 chars, col should clamp to 2
	if row != 0 || col != 2 {
		t.Errorf("after up with col clamp: CursorPos() = (%d, %d); want (0, 2)", row, col)
	}
}

func TestEditorModel_BackspaceAtVeryStart(t *testing.T) {
	m := NewEditorModel()
	// Backspace on empty editor: should be no-op
	updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyBackspace})
	m = updated.(EditorModel)
	if got := m.Text(); got != "" {
		t.Errorf("backspace on empty: Text() = %q; want empty", got)
	}
}

func TestEditorModel_KillAtEndOfLine(t *testing.T) {
	m := NewEditorModel()
	m = m.SetText("hello")
	// Cursor at end: kill should be no-op
	updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyCtrlK})
	m = updated.(EditorModel)
	if got := m.Text(); got != "hello" {
		t.Errorf("kill at end: Text() = %q; want %q", got, "hello")
	}
}

// --- Ghost text tests ---

func TestEditorModel_SetGhostText(t *testing.T) {
	t.Parallel()

	m := NewEditorModel()
	m = m.SetGhostText("lp")
	if got := m.GhostText(); got != "lp" {
		t.Errorf("GhostText() = %q; want %q", got, "lp")
	}
}

func TestEditorModel_GhostTextDefaultEmpty(t *testing.T) {
	t.Parallel()

	m := NewEditorModel()
	if got := m.GhostText(); got != "" {
		t.Errorf("expected empty ghost text, got %q", got)
	}
}

func TestEditorModel_AcceptGhostText(t *testing.T) {
	t.Parallel()

	m := NewEditorModel()
	m = m.SetText("/he")
	m = m.SetGhostText("lp")

	// Tab should accept ghost text
	updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyTab})
	m = updated.(EditorModel)

	if got := m.Text(); got != "/help" {
		t.Errorf("after accept: Text() = %q; want %q", got, "/help")
	}
	if got := m.GhostText(); got != "" {
		t.Errorf("ghost text should be cleared after accept, got %q", got)
	}
}

func TestEditorModel_TabWithoutGhostText(t *testing.T) {
	t.Parallel()

	m := NewEditorModel()
	m = m.SetText("hello")

	// Tab with no ghost text: inserts tab character
	updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyTab})
	m = updated.(EditorModel)

	if got := m.Text(); got != "hello\t" {
		t.Errorf("tab without ghost text should insert tab, got %q", got)
	}
}

func TestEditorModel_GhostTextInView(t *testing.T) {
	t.Parallel()

	m := NewEditorModel()
	m.width = 80
	m = m.SetFocused(true)
	m = m.SetText("/he")
	m = m.SetGhostText("lp")

	view := m.View()
	if !strings.Contains(view, "lp") {
		t.Errorf("expected ghost text 'lp' in view, got:\n%s", view)
	}
}
