// ABOUTME: Tests for the multi-line text editor component
// ABOUTME: Covers typing, cursor movement, word-wrap, undo/redo, kill ring, focus

package component

import (
	"strings"
	"testing"

	"github.com/mauromedda/pi-coding-agent-go/pkg/tui"
	"github.com/mauromedda/pi-coding-agent-go/pkg/tui/key"
)

func TestEditor_NewEditor(t *testing.T) {
	t.Parallel()

	ed := NewEditor()
	if ed.Text() != "" {
		t.Errorf("expected empty text, got %q", ed.Text())
	}
	row, col := ed.CursorPos()
	if row != 0 || col != 0 {
		t.Errorf("expected cursor at (0,0), got (%d,%d)", row, col)
	}
}

func TestEditor_TypeCharacters(t *testing.T) {
	t.Parallel()

	ed := NewEditor()
	ed.SetFocused(true)
	ed.HandleInput("H")
	ed.HandleInput("i")

	if ed.Text() != "Hi" {
		t.Errorf("expected 'Hi', got %q", ed.Text())
	}
	row, col := ed.CursorPos()
	if row != 0 || col != 2 {
		t.Errorf("expected cursor at (0,2), got (%d,%d)", row, col)
	}
}

func TestEditor_Enter(t *testing.T) {
	t.Parallel()

	ed := NewEditor()
	ed.SetFocused(true)
	ed.HandleInput("a")
	ed.HandleInput("b")
	ed.HandleInput("\r") // enter
	ed.HandleInput("c")

	if ed.Text() != "ab\nc" {
		t.Errorf("expected 'ab\\nc', got %q", ed.Text())
	}
	row, col := ed.CursorPos()
	if row != 1 || col != 1 {
		t.Errorf("expected cursor at (1,1), got (%d,%d)", row, col)
	}
}

func TestEditor_Backspace(t *testing.T) {
	t.Parallel()

	ed := NewEditor()
	ed.SetFocused(true)
	ed.HandleInput("a")
	ed.HandleInput("b")
	ed.HandleInput("\x7f") // backspace

	if ed.Text() != "a" {
		t.Errorf("expected 'a', got %q", ed.Text())
	}
}

func TestEditor_BackspaceJoinsLines(t *testing.T) {
	t.Parallel()

	ed := NewEditor()
	ed.SetFocused(true)
	ed.HandleInput("a")
	ed.HandleInput("\r") // enter
	ed.HandleInput("b")
	// Cursor at (1,1). Move to start of line 1
	ed.HandleInput("\x1b[H") // home -> (1,0)
	ed.HandleInput("\x7f")   // backspace should join with previous line

	if ed.Text() != "ab" {
		t.Errorf("expected 'ab', got %q", ed.Text())
	}
	row, col := ed.CursorPos()
	if row != 0 || col != 1 {
		t.Errorf("expected cursor at (0,1), got (%d,%d)", row, col)
	}
}

func TestEditor_BackspaceAtStart(t *testing.T) {
	t.Parallel()

	ed := NewEditor()
	ed.SetFocused(true)
	ed.HandleInput("\x7f") // backspace on empty

	if ed.Text() != "" {
		t.Errorf("expected empty, got %q", ed.Text())
	}
}

func TestEditor_Delete(t *testing.T) {
	t.Parallel()

	ed := NewEditor()
	ed.SetFocused(true)
	ed.HandleInput("a")
	ed.HandleInput("b")
	ed.HandleInput("\x1b[D")  // left
	ed.HandleInput("\x1b[D")  // left
	ed.HandleInput("\x1b[3~") // delete

	if ed.Text() != "b" {
		t.Errorf("expected 'b', got %q", ed.Text())
	}
}

func TestEditor_DeleteJoinsLines(t *testing.T) {
	t.Parallel()

	ed := NewEditor()
	ed.SetFocused(true)
	ed.HandleInput("a")
	ed.HandleInput("\r") // enter
	ed.HandleInput("b")
	// Move to end of line 0
	ed.HandleInput("\x1b[A")  // up
	ed.HandleInput("\x1b[F")  // end
	ed.HandleInput("\x1b[3~") // delete at end of first line joins

	if ed.Text() != "ab" {
		t.Errorf("expected 'ab', got %q", ed.Text())
	}
}

func TestEditor_ArrowUpDown(t *testing.T) {
	t.Parallel()

	ed := NewEditor()
	ed.SetFocused(true)
	ed.HandleInput("a")
	ed.HandleInput("b")
	ed.HandleInput("c")
	ed.HandleInput("\r")
	ed.HandleInput("d")
	ed.HandleInput("e")

	row, _ := ed.CursorPos()
	if row != 1 {
		t.Errorf("expected row 1, got %d", row)
	}

	ed.HandleInput("\x1b[A") // up
	row, _ = ed.CursorPos()
	if row != 0 {
		t.Errorf("expected row 0 after up, got %d", row)
	}

	ed.HandleInput("\x1b[B") // down
	row, _ = ed.CursorPos()
	if row != 1 {
		t.Errorf("expected row 1 after down, got %d", row)
	}
}

func TestEditor_ArrowLeftRight(t *testing.T) {
	t.Parallel()

	ed := NewEditor()
	ed.SetFocused(true)
	ed.HandleInput("a")
	ed.HandleInput("b")

	ed.HandleInput("\x1b[D") // left
	_, col := ed.CursorPos()
	if col != 1 {
		t.Errorf("expected col 1, got %d", col)
	}

	ed.HandleInput("\x1b[C") // right
	_, col = ed.CursorPos()
	if col != 2 {
		t.Errorf("expected col 2, got %d", col)
	}
}

func TestEditor_HomeEnd(t *testing.T) {
	t.Parallel()

	ed := NewEditor()
	ed.SetFocused(true)
	ed.HandleInput("h")
	ed.HandleInput("e")
	ed.HandleInput("l")
	ed.HandleInput("l")
	ed.HandleInput("o")

	ed.HandleInput("\x1b[H") // home
	_, col := ed.CursorPos()
	if col != 0 {
		t.Errorf("expected col 0 after Home, got %d", col)
	}

	ed.HandleInput("\x1b[F") // end
	_, col = ed.CursorPos()
	if col != 5 {
		t.Errorf("expected col 5 after End, got %d", col)
	}
}

func TestEditor_KillLine(t *testing.T) {
	t.Parallel()

	ed := NewEditor()
	ed.SetFocused(true)
	for _, ch := range "hello" {
		ed.HandleInput(string(ch))
	}
	ed.HandleInput("\x1b[H") // home
	ed.HandleInput("\x1b[C") // right -> col 1
	ed.HandleInput("\x0b")   // Ctrl+K

	if ed.Text() != "h" {
		t.Errorf("expected 'h', got %q", ed.Text())
	}
}

func TestEditor_Yank(t *testing.T) {
	t.Parallel()

	ed := NewEditor()
	ed.SetFocused(true)
	for _, ch := range "hello" {
		ed.HandleInput(string(ch))
	}
	ed.HandleInput("\x1b[H") // home
	ed.HandleInput("\x1b[C") // right -> col 1
	ed.HandleInput("\x0b")   // Ctrl+K -> kills "ello"
	ed.HandleInput("\x19")   // Ctrl+Y -> yank

	if ed.Text() != "hello" {
		t.Errorf("expected 'hello', got %q", ed.Text())
	}
}

func TestEditor_Undo(t *testing.T) {
	t.Parallel()

	ed := NewEditor()
	ed.SetFocused(true)
	ed.HandleInput("a")
	ed.HandleInput("b")
	ed.HandleInput("c")
	ed.HandleInput("\x1a") // Ctrl+Z = undo

	if ed.Text() != "ab" {
		t.Errorf("expected 'ab' after undo, got %q", ed.Text())
	}
}

func TestEditor_SetText(t *testing.T) {
	t.Parallel()

	ed := NewEditor()
	ed.SetText("line1\nline2\nline3")

	if ed.Text() != "line1\nline2\nline3" {
		t.Errorf("expected three lines, got %q", ed.Text())
	}
}

func TestEditor_Focus(t *testing.T) {
	t.Parallel()

	ed := NewEditor()
	if ed.IsFocused() {
		t.Error("expected not focused initially")
	}
	ed.SetFocused(true)
	if !ed.IsFocused() {
		t.Error("expected focused after SetFocused(true)")
	}
}

func TestEditor_RenderBasic(t *testing.T) {
	t.Parallel()

	ed := NewEditor()
	ed.SetFocused(true)
	for _, ch := range "hello" {
		ed.HandleInput(string(ch))
	}
	ed.HandleInput("\r")
	for _, ch := range "world" {
		ed.HandleInput(string(ch))
	}

	buf := tui.AcquireBuffer()
	defer tui.ReleaseBuffer(buf)

	ed.Render(buf, 40)

	if buf.Len() < 2 {
		t.Fatalf("expected at least 2 lines, got %d", buf.Len())
	}
}

func TestEditor_RenderWithCursorMarker(t *testing.T) {
	t.Parallel()

	ed := NewEditor()
	ed.SetFocused(true)
	ed.HandleInput("a")

	buf := tui.AcquireBuffer()
	defer tui.ReleaseBuffer(buf)

	ed.Render(buf, 40)

	found := false
	for _, line := range buf.Lines {
		if strings.Contains(line, tui.CursorMarker) {
			found = true
			break
		}
	}
	if !found {
		t.Error("expected cursor marker in rendered output")
	}
}

func TestEditor_RenderNoCursorWhenUnfocused(t *testing.T) {
	t.Parallel()

	ed := NewEditor()
	ed.HandleInput("a")

	buf := tui.AcquireBuffer()
	defer tui.ReleaseBuffer(buf)

	ed.Render(buf, 40)

	for _, line := range buf.Lines {
		if strings.Contains(line, tui.CursorMarker) {
			t.Error("expected no cursor marker when unfocused")
			break
		}
	}
}

func TestEditor_RenderWordWrap(t *testing.T) {
	t.Parallel()

	ed := NewEditor()
	ed.SetFocused(true)
	// Type a string that exceeds 10 columns
	for _, ch := range "abcdefghijklmno" {
		ed.HandleInput(string(ch))
	}

	buf := tui.AcquireBuffer()
	defer tui.ReleaseBuffer(buf)

	ed.Render(buf, 10)

	// With width=10, 15 chars should wrap to at least 2 lines
	if buf.Len() < 2 {
		t.Errorf("expected word-wrap to produce >=2 lines for 15 chars at width 10, got %d", buf.Len())
	}
}

func TestEditor_Invalidate(t *testing.T) {
	t.Parallel()

	ed := NewEditor()
	ed.HandleInput("test")
	ed.Invalidate()

	buf := tui.AcquireBuffer()
	defer tui.ReleaseBuffer(buf)

	ed.Render(buf, 40)
	if buf.Len() < 1 {
		t.Fatal("expected at least 1 line after invalidate")
	}
}

func TestEditor_MultilineNavigation(t *testing.T) {
	t.Parallel()

	ed := NewEditor()
	ed.SetFocused(true)
	// Line 0: "abc"
	ed.HandleInput("a")
	ed.HandleInput("b")
	ed.HandleInput("c")
	// Line 1: "de"
	ed.HandleInput("\r")
	ed.HandleInput("d")
	ed.HandleInput("e")

	// Go up: cursor should clamp to shorter line length
	ed.HandleInput("\x1b[A") // up -> row 0
	row, col := ed.CursorPos()
	if row != 0 {
		t.Errorf("expected row 0, got %d", row)
	}
	if col > 3 {
		t.Errorf("expected col <= 3, got %d", col)
	}
}

func TestEditor_InsertAtMiddleOfLine(t *testing.T) {
	t.Parallel()

	ed := NewEditor()
	ed.SetFocused(true)
	ed.HandleInput("a")
	ed.HandleInput("c")
	ed.HandleInput("\x1b[D") // left
	ed.HandleInput("b")

	if ed.Text() != "abc" {
		t.Errorf("expected 'abc', got %q", ed.Text())
	}
}

func TestEditor_EnterSplitsLine(t *testing.T) {
	t.Parallel()

	ed := NewEditor()
	ed.SetFocused(true)
	ed.HandleInput("a")
	ed.HandleInput("b")
	ed.HandleInput("c")
	ed.HandleInput("\x1b[D") // left -> cursor at col 2 ("ab|c")
	ed.HandleInput("\r")     // enter splits line

	if ed.Text() != "ab\nc" {
		t.Errorf("expected 'ab\\nc', got %q", ed.Text())
	}
	row, col := ed.CursorPos()
	if row != 1 || col != 0 {
		t.Errorf("expected cursor at (1,0), got (%d,%d)", row, col)
	}
}

func TestEditor_HandleKey_Rune(t *testing.T) {
	t.Parallel()

	ed := NewEditor()
	ed.SetFocused(true)

	ed.HandleKey(key.Key{Type: key.KeyRune, Rune: 'H'})
	ed.HandleKey(key.Key{Type: key.KeyRune, Rune: 'i'})

	if ed.Text() != "Hi" {
		t.Errorf("expected 'Hi', got %q", ed.Text())
	}
	row, col := ed.CursorPos()
	if row != 0 || col != 2 {
		t.Errorf("expected cursor at (0,2), got (%d,%d)", row, col)
	}
}

func TestEditor_HandleKey_Enter(t *testing.T) {
	t.Parallel()

	ed := NewEditor()
	ed.SetFocused(true)
	ed.HandleKey(key.Key{Type: key.KeyRune, Rune: 'a'})
	ed.HandleKey(key.Key{Type: key.KeyEnter})
	ed.HandleKey(key.Key{Type: key.KeyRune, Rune: 'b'})

	if ed.Text() != "a\nb" {
		t.Errorf("expected 'a\\nb', got %q", ed.Text())
	}
}

func TestEditor_HandleKey_Backspace(t *testing.T) {
	t.Parallel()

	ed := NewEditor()
	ed.SetFocused(true)
	ed.HandleKey(key.Key{Type: key.KeyRune, Rune: 'x'})
	ed.HandleKey(key.Key{Type: key.KeyRune, Rune: 'y'})
	ed.HandleKey(key.Key{Type: key.KeyBackspace})

	if ed.Text() != "x" {
		t.Errorf("expected 'x', got %q", ed.Text())
	}
}

func TestEditor_HandleKey_Navigation(t *testing.T) {
	t.Parallel()

	ed := NewEditor()
	ed.SetFocused(true)
	ed.HandleKey(key.Key{Type: key.KeyRune, Rune: 'a'})
	ed.HandleKey(key.Key{Type: key.KeyRune, Rune: 'b'})
	ed.HandleKey(key.Key{Type: key.KeyLeft})
	ed.HandleKey(key.Key{Type: key.KeyLeft})
	ed.HandleKey(key.Key{Type: key.KeyRight})

	_, col := ed.CursorPos()
	if col != 1 {
		t.Errorf("expected col 1, got %d", col)
	}
}

func TestEditor_HandleKey_HomeEnd(t *testing.T) {
	t.Parallel()

	ed := NewEditor()
	ed.SetFocused(true)
	ed.HandleKey(key.Key{Type: key.KeyRune, Rune: 'a'})
	ed.HandleKey(key.Key{Type: key.KeyRune, Rune: 'b'})
	ed.HandleKey(key.Key{Type: key.KeyRune, Rune: 'c'})
	ed.HandleKey(key.Key{Type: key.KeyHome})

	_, col := ed.CursorPos()
	if col != 0 {
		t.Errorf("expected col 0 after Home, got %d", col)
	}

	ed.HandleKey(key.Key{Type: key.KeyEnd})
	_, col = ed.CursorPos()
	if col != 3 {
		t.Errorf("expected col 3 after End, got %d", col)
	}
}

func TestEditor_HandleKey_CtrlA_CtrlE(t *testing.T) {
	t.Parallel()

	ed := NewEditor()
	ed.SetFocused(true)
	ed.HandleKey(key.Key{Type: key.KeyRune, Rune: 'x'})
	ed.HandleKey(key.Key{Type: key.KeyRune, Rune: 'y'})

	// Ctrl+A = home
	ed.HandleKey(key.Key{Type: key.KeyRune, Rune: 'a', Ctrl: true})
	_, col := ed.CursorPos()
	if col != 0 {
		t.Errorf("expected col 0 after Ctrl+A, got %d", col)
	}

	// Ctrl+E = end
	ed.HandleKey(key.Key{Type: key.KeyRune, Rune: 'e', Ctrl: true})
	_, col = ed.CursorPos()
	if col != 2 {
		t.Errorf("expected col 2 after Ctrl+E, got %d", col)
	}
}

func TestEditor_PromptPrefix(t *testing.T) {
	t.Parallel()

	ed := NewEditor()
	ed.SetFocused(true)
	ed.SetPrompt("❯ ")
	ed.HandleKey(key.Key{Type: key.KeyRune, Rune: 'h'})
	ed.HandleKey(key.Key{Type: key.KeyRune, Rune: 'i'})

	buf := tui.AcquireBuffer()
	defer tui.ReleaseBuffer(buf)

	ed.Render(buf, 40)

	if buf.Len() < 1 {
		t.Fatal("expected at least 1 line")
	}
	line0 := buf.Lines[0]
	// Line 0 must start with the prompt prefix (before cursor marker)
	stripped := stripCursorMarker(line0)
	if !strings.HasPrefix(stripped, "❯ ") {
		t.Errorf("expected line0 to start with prompt, got %q", stripped)
	}
	if !strings.Contains(stripped, "hi") {
		t.Errorf("expected line0 to contain 'hi', got %q", stripped)
	}
}

func TestEditor_PromptCursorOffset(t *testing.T) {
	t.Parallel()

	ed := NewEditor()
	ed.SetFocused(true)
	ed.SetPrompt("❯ ")
	// Empty editor: cursor should be at prompt position
	buf := tui.AcquireBuffer()
	defer tui.ReleaseBuffer(buf)

	ed.Render(buf, 40)

	if buf.Len() < 1 {
		t.Fatal("expected at least 1 line")
	}
	// CursorMarker should be present after the prompt
	line0 := buf.Lines[0]
	if !strings.Contains(line0, tui.CursorMarker) {
		t.Error("expected cursor marker in rendered output")
	}
	// Cursor marker should come after the prompt
	before, _, _ := strings.Cut(line0, tui.CursorMarker)
	beforeMarker := before
	if !strings.Contains(beforeMarker, "❯") {
		t.Errorf("expected prompt before cursor marker, got %q", beforeMarker)
	}
}

func TestEditor_PromptMultilineIndent(t *testing.T) {
	t.Parallel()

	ed := NewEditor()
	ed.SetFocused(true)
	ed.SetPrompt("❯ ")
	for _, ch := range "line1" {
		ed.HandleKey(key.Key{Type: key.KeyRune, Rune: ch})
	}
	ed.HandleKey(key.Key{Type: key.KeyEnter})
	for _, ch := range "line2" {
		ed.HandleKey(key.Key{Type: key.KeyRune, Rune: ch})
	}

	buf := tui.AcquireBuffer()
	defer tui.ReleaseBuffer(buf)

	ed.Render(buf, 40)

	if buf.Len() < 2 {
		t.Fatalf("expected at least 2 lines, got %d", buf.Len())
	}
	// Line 0 should have the prompt prefix
	stripped0 := stripCursorMarker(buf.Lines[0])
	if !strings.HasPrefix(stripped0, "❯ ") {
		t.Errorf("expected line0 prompt prefix, got %q", stripped0)
	}
	// Line 1 should have indent matching prompt width (2 visible chars for "❯ ")
	stripped1 := stripCursorMarker(buf.Lines[1])
	if !strings.HasPrefix(stripped1, "  ") {
		t.Errorf("expected line1 indent of 2 spaces, got %q", stripped1)
	}
	if !strings.Contains(stripped1, "line2") {
		t.Errorf("expected line1 to contain 'line2', got %q", stripped1)
	}
}

func TestEditor_PromptWrapWidthReduced(t *testing.T) {
	t.Parallel()

	ed := NewEditor()
	ed.SetFocused(true)
	ed.SetPrompt("❯ ")
	// Type 10 chars; with prompt width=2 and terminal width=8, effective=6
	// So 10 chars should wrap
	for _, ch := range "abcdefghij" {
		ed.HandleInput(string(ch))
	}

	buf := tui.AcquireBuffer()
	defer tui.ReleaseBuffer(buf)

	ed.Render(buf, 8)

	// With effective width 6, 10 chars need ceil(10/6) = 2 wrapped lines
	if buf.Len() < 2 {
		t.Errorf("expected wrapping with reduced width, got %d lines", buf.Len())
	}
}

func TestEditor_PlaceholderShownWhenEmptyFocused(t *testing.T) {
	t.Parallel()

	ed := NewEditor()
	ed.SetFocused(true)
	ed.SetPrompt("❯ ")
	ed.SetPlaceholder("Type something...")

	buf := tui.AcquireBuffer()
	defer tui.ReleaseBuffer(buf)

	ed.Render(buf, 40)

	if buf.Len() < 1 {
		t.Fatal("expected at least 1 line")
	}
	line0 := buf.Lines[0]
	// Should contain the dim placeholder text
	if !strings.Contains(line0, "Type something...") {
		t.Errorf("expected placeholder text, got %q", line0)
	}
	// Should contain dim ANSI
	if !strings.Contains(line0, "\x1b[2m") {
		t.Errorf("expected dim ANSI for placeholder, got %q", line0)
	}
}

func TestEditor_PlaceholderHiddenWhenText(t *testing.T) {
	t.Parallel()

	ed := NewEditor()
	ed.SetFocused(true)
	ed.SetPrompt("❯ ")
	ed.SetPlaceholder("Type something...")
	ed.HandleKey(key.Key{Type: key.KeyRune, Rune: 'h'})

	buf := tui.AcquireBuffer()
	defer tui.ReleaseBuffer(buf)

	ed.Render(buf, 40)

	for _, line := range buf.Lines {
		if strings.Contains(line, "Type something...") {
			t.Error("placeholder should not appear when editor has text")
		}
	}
}

func TestEditor_PlaceholderHiddenWhenUnfocused(t *testing.T) {
	t.Parallel()

	ed := NewEditor()
	ed.SetPlaceholder("Type something...")
	// unfocused by default

	buf := tui.AcquireBuffer()
	defer tui.ReleaseBuffer(buf)

	ed.Render(buf, 40)

	for _, line := range buf.Lines {
		if strings.Contains(line, "Type something...") {
			t.Error("placeholder should not appear when unfocused")
		}
	}
}

func TestEditor_NoPromptBackwardCompat(t *testing.T) {
	t.Parallel()

	ed := NewEditor()
	ed.SetFocused(true)
	for _, ch := range "test" {
		ed.HandleKey(key.Key{Type: key.KeyRune, Rune: ch})
	}

	buf := tui.AcquireBuffer()
	defer tui.ReleaseBuffer(buf)

	ed.Render(buf, 40)

	if buf.Len() < 1 {
		t.Fatal("expected at least 1 line")
	}
	// Without prompt set, line should not have any prefix
	stripped := stripCursorMarker(buf.Lines[0])
	if strings.HasPrefix(stripped, "❯") || strings.HasPrefix(stripped, " ") {
		t.Errorf("expected no prefix without SetPrompt, got %q", stripped)
	}
	if !strings.Contains(stripped, "test") {
		t.Errorf("expected 'test' in output, got %q", stripped)
	}
}

// stripCursorMarker removes the cursor marker from a string for easier assertion.
func stripCursorMarker(s string) string {
	return strings.ReplaceAll(s, tui.CursorMarker, "")
}

func TestEditor_HandleKey_ConcurrentSafety(t *testing.T) {
	t.Parallel()

	ed := NewEditor()
	ed.SetFocused(true)

	done := make(chan struct{})
	go func() {
		defer close(done)
		for range 100 {
			ed.HandleKey(key.Key{Type: key.KeyRune, Rune: 'a'})
		}
	}()

	// Concurrent renders
	buf := tui.AcquireBuffer()
	defer tui.ReleaseBuffer(buf)
	for range 100 {
		buf.Lines = buf.Lines[:0]
		ed.Render(buf, 80)
	}

	<-done

	text := ed.Text()
	if len(text) != 100 {
		t.Errorf("expected 100 'a's, got %d chars: %q", len(text), text)
	}
}
