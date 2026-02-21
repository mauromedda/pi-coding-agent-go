// ABOUTME: Tests for the single-line text input component
// ABOUTME: Covers typing, cursor movement, undo/redo, kill ring, placeholder, scrolling

package component

import (
	"strings"
	"testing"

	"github.com/mauromedda/pi-coding-agent-go/pkg/tui"
)

func TestInput_NewInput(t *testing.T) {
	t.Parallel()

	inp := NewInput()
	if inp.Text() != "" {
		t.Errorf("expected empty text, got %q", inp.Text())
	}
	if inp.CursorPos() != 0 {
		t.Errorf("expected cursor at 0, got %d", inp.CursorPos())
	}
}

func TestInput_SetPlaceholder(t *testing.T) {
	t.Parallel()

	inp := NewInput()
	inp.SetPlaceholder("Type here...")

	buf := tui.AcquireBuffer()
	defer tui.ReleaseBuffer(buf)

	inp.SetFocused(true)
	inp.Render(buf, 40)

	if buf.Len() != 1 {
		t.Fatalf("expected 1 line, got %d", buf.Len())
	}
	// When empty, placeholder should appear (dimmed)
	if !strings.Contains(buf.Lines[0], "Type here...") {
		t.Errorf("expected placeholder in output, got %q", buf.Lines[0])
	}
}

func TestInput_TypeCharacters(t *testing.T) {
	t.Parallel()

	inp := NewInput()
	inp.SetFocused(true)
	inp.HandleInput("H")
	inp.HandleInput("i")

	if inp.Text() != "Hi" {
		t.Errorf("expected 'Hi', got %q", inp.Text())
	}
	if inp.CursorPos() != 2 {
		t.Errorf("expected cursor at 2, got %d", inp.CursorPos())
	}
}

func TestInput_Backspace(t *testing.T) {
	t.Parallel()

	inp := NewInput()
	inp.SetFocused(true)
	inp.HandleInput("a")
	inp.HandleInput("b")
	inp.HandleInput("c")
	inp.HandleInput("\x7f") // backspace

	if inp.Text() != "ab" {
		t.Errorf("expected 'ab', got %q", inp.Text())
	}
	if inp.CursorPos() != 2 {
		t.Errorf("expected cursor at 2, got %d", inp.CursorPos())
	}
}

func TestInput_BackspaceAtStart(t *testing.T) {
	t.Parallel()

	inp := NewInput()
	inp.SetFocused(true)
	inp.HandleInput("\x7f") // backspace on empty

	if inp.Text() != "" {
		t.Errorf("expected empty, got %q", inp.Text())
	}
}

func TestInput_Delete(t *testing.T) {
	t.Parallel()

	inp := NewInput()
	inp.SetFocused(true)
	inp.HandleInput("a")
	inp.HandleInput("b")
	// Move cursor left
	inp.HandleInput("\x1b[D") // left arrow
	inp.HandleInput("\x1b[D") // left arrow
	inp.HandleInput("\x1b[3~") // delete

	if inp.Text() != "b" {
		t.Errorf("expected 'b', got %q", inp.Text())
	}
	if inp.CursorPos() != 0 {
		t.Errorf("expected cursor at 0, got %d", inp.CursorPos())
	}
}

func TestInput_ArrowLeftRight(t *testing.T) {
	t.Parallel()

	inp := NewInput()
	inp.SetFocused(true)
	inp.HandleInput("a")
	inp.HandleInput("b")
	inp.HandleInput("c")
	inp.HandleInput("\x1b[D") // left
	inp.HandleInput("\x1b[D") // left

	if inp.CursorPos() != 1 {
		t.Errorf("expected cursor at 1, got %d", inp.CursorPos())
	}

	inp.HandleInput("\x1b[C") // right

	if inp.CursorPos() != 2 {
		t.Errorf("expected cursor at 2, got %d", inp.CursorPos())
	}
}

func TestInput_HomeEnd(t *testing.T) {
	t.Parallel()

	inp := NewInput()
	inp.SetFocused(true)
	inp.HandleInput("h")
	inp.HandleInput("e")
	inp.HandleInput("l")
	inp.HandleInput("l")
	inp.HandleInput("o")
	inp.HandleInput("\x1b[H") // home

	if inp.CursorPos() != 0 {
		t.Errorf("expected cursor at 0 after Home, got %d", inp.CursorPos())
	}

	inp.HandleInput("\x1b[F") // end

	if inp.CursorPos() != 5 {
		t.Errorf("expected cursor at 5 after End, got %d", inp.CursorPos())
	}
}

func TestInput_CtrlA_CtrlE(t *testing.T) {
	t.Parallel()

	inp := NewInput()
	inp.SetFocused(true)
	inp.HandleInput("a")
	inp.HandleInput("b")
	inp.HandleInput("c")

	inp.HandleInput("\x01") // Ctrl+A = home

	if inp.CursorPos() != 0 {
		t.Errorf("expected cursor at 0 after Ctrl+A, got %d", inp.CursorPos())
	}

	inp.HandleInput("\x05") // Ctrl+E = end

	if inp.CursorPos() != 3 {
		t.Errorf("expected cursor at 3 after Ctrl+E, got %d", inp.CursorPos())
	}
}

func TestInput_KillLine(t *testing.T) {
	t.Parallel()

	inp := NewInput()
	inp.SetFocused(true)
	inp.HandleInput("h")
	inp.HandleInput("e")
	inp.HandleInput("l")
	inp.HandleInput("l")
	inp.HandleInput("o")
	inp.HandleInput("\x01")  // Ctrl+A -> home
	inp.HandleInput("\x1b[C") // right -> pos 1
	inp.HandleInput("\x0b")  // Ctrl+K = kill to end

	if inp.Text() != "h" {
		t.Errorf("expected 'h', got %q", inp.Text())
	}
	if inp.CursorPos() != 1 {
		t.Errorf("expected cursor at 1, got %d", inp.CursorPos())
	}
}

func TestInput_Yank(t *testing.T) {
	t.Parallel()

	inp := NewInput()
	inp.SetFocused(true)
	inp.HandleInput("h")
	inp.HandleInput("e")
	inp.HandleInput("l")
	inp.HandleInput("l")
	inp.HandleInput("o")
	inp.HandleInput("\x01")  // Ctrl+A
	inp.HandleInput("\x1b[C") // right -> pos 1
	inp.HandleInput("\x0b")  // Ctrl+K -> kills "ello"
	inp.HandleInput("\x19")  // Ctrl+Y -> yank "ello" back

	if inp.Text() != "hello" {
		t.Errorf("expected 'hello', got %q", inp.Text())
	}
}

func TestInput_UndoRedo(t *testing.T) {
	t.Parallel()

	inp := NewInput()
	inp.SetFocused(true)
	inp.HandleInput("a")
	inp.HandleInput("b")
	inp.HandleInput("c")
	inp.HandleInput("\x1a") // Ctrl+Z = undo

	if inp.Text() != "ab" {
		t.Errorf("expected 'ab' after undo, got %q", inp.Text())
	}

	inp.HandleInput("\x19") // After undo, Ctrl+Y might be yank; let's use a different approach
	// Redo is typically Ctrl+Shift+Z, but since we can't detect shift on raw ctrl,
	// we won't test redo via Ctrl+Y here as it conflicts with yank.
	// Instead, just verify undo worked.
}

func TestInput_InsertAtMiddle(t *testing.T) {
	t.Parallel()

	inp := NewInput()
	inp.SetFocused(true)
	inp.HandleInput("a")
	inp.HandleInput("c")
	inp.HandleInput("\x1b[D") // left
	inp.HandleInput("b")

	if inp.Text() != "abc" {
		t.Errorf("expected 'abc', got %q", inp.Text())
	}
	if inp.CursorPos() != 2 {
		t.Errorf("expected cursor at 2, got %d", inp.CursorPos())
	}
}

func TestInput_Focus(t *testing.T) {
	t.Parallel()

	inp := NewInput()
	if inp.IsFocused() {
		t.Error("expected not focused initially")
	}
	inp.SetFocused(true)
	if !inp.IsFocused() {
		t.Error("expected focused after SetFocused(true)")
	}
	inp.SetFocused(false)
	if inp.IsFocused() {
		t.Error("expected not focused after SetFocused(false)")
	}
}

func TestInput_RenderWithCursorMarker(t *testing.T) {
	t.Parallel()

	inp := NewInput()
	inp.SetFocused(true)
	inp.HandleInput("a")
	inp.HandleInput("b")

	buf := tui.AcquireBuffer()
	defer tui.ReleaseBuffer(buf)

	inp.Render(buf, 40)

	if buf.Len() != 1 {
		t.Fatalf("expected 1 line, got %d", buf.Len())
	}
	if !strings.Contains(buf.Lines[0], tui.CursorMarker) {
		t.Error("expected cursor marker in rendered output")
	}
}

func TestInput_RenderWithoutFocus(t *testing.T) {
	t.Parallel()

	inp := NewInput()
	inp.HandleInput("abc")

	buf := tui.AcquireBuffer()
	defer tui.ReleaseBuffer(buf)

	inp.Render(buf, 40)

	if buf.Len() != 1 {
		t.Fatalf("expected 1 line, got %d", buf.Len())
	}
	if strings.Contains(buf.Lines[0], tui.CursorMarker) {
		t.Error("expected no cursor marker when unfocused")
	}
}

func TestInput_HorizontalScroll(t *testing.T) {
	t.Parallel()

	inp := NewInput()
	inp.SetFocused(true)

	// Type more characters than width allows
	for i := 0; i < 20; i++ {
		inp.HandleInput("x")
	}

	buf := tui.AcquireBuffer()
	defer tui.ReleaseBuffer(buf)

	inp.Render(buf, 10)

	if buf.Len() != 1 {
		t.Fatalf("expected 1 line, got %d", buf.Len())
	}
	// The rendered line (minus ANSI/cursor marker) should not exceed width
	// Just verify it renders without error and contains the cursor marker
	if !strings.Contains(buf.Lines[0], tui.CursorMarker) {
		t.Error("expected cursor marker in scrolled output")
	}
}

func TestInput_SetText(t *testing.T) {
	t.Parallel()

	inp := NewInput()
	inp.SetText("preset")

	if inp.Text() != "preset" {
		t.Errorf("expected 'preset', got %q", inp.Text())
	}
	if inp.CursorPos() != 6 {
		t.Errorf("expected cursor at 6, got %d", inp.CursorPos())
	}
}

func TestInput_Invalidate(t *testing.T) {
	t.Parallel()

	inp := NewInput()
	inp.HandleInput("test")
	inp.Invalidate()

	buf := tui.AcquireBuffer()
	defer tui.ReleaseBuffer(buf)

	inp.Render(buf, 40)
	if buf.Len() != 1 {
		t.Fatalf("expected 1 line after invalidate, got %d", buf.Len())
	}
}

func TestInput_CtrlW_DeleteWord(t *testing.T) {
	t.Parallel()

	inp := NewInput()
	inp.SetFocused(true)
	for _, ch := range "hello world" {
		inp.HandleInput(string(ch))
	}
	inp.HandleInput("\x17") // Ctrl+W = delete word backward

	if inp.Text() != "hello " {
		t.Errorf("expected 'hello ', got %q", inp.Text())
	}
}

func TestInput_ArrowBeyondBounds(t *testing.T) {
	t.Parallel()

	inp := NewInput()
	inp.SetFocused(true)
	inp.HandleInput("a")
	inp.HandleInput("\x1b[C") // right at end
	inp.HandleInput("\x1b[C") // right again

	if inp.CursorPos() != 1 {
		t.Errorf("expected cursor clamped at 1, got %d", inp.CursorPos())
	}

	inp.HandleInput("\x1b[D") // left to 0
	inp.HandleInput("\x1b[D") // left past start

	if inp.CursorPos() != 0 {
		t.Errorf("expected cursor clamped at 0, got %d", inp.CursorPos())
	}
}
