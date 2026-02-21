// ABOUTME: Tests for vim mode in the single-line text input component
// ABOUTME: Covers enable/disable, normal mode navigation, insert transitions, delete, change, word motions

package component

import (
	"testing"
)

func TestVim_DisabledByDefault(t *testing.T) {
	t.Parallel()

	inp := NewInput()
	if inp.VimEnabled() {
		t.Error("expected vim disabled by default")
	}
	if inp.VimMode() != VimInsert {
		t.Errorf("expected VimInsert mode by default, got %d", inp.VimMode())
	}
}

func TestVim_EnableDisable(t *testing.T) {
	t.Parallel()

	inp := NewInput()
	inp.SetVimEnabled(true)
	if !inp.VimEnabled() {
		t.Error("expected vim enabled after SetVimEnabled(true)")
	}
	if inp.VimMode() != VimNormal {
		t.Errorf("expected VimNormal after enabling, got %d", inp.VimMode())
	}

	inp.SetVimEnabled(false)
	if inp.VimEnabled() {
		t.Error("expected vim disabled after SetVimEnabled(false)")
	}
	if inp.VimMode() != VimInsert {
		t.Errorf("expected VimInsert after disabling, got %d", inp.VimMode())
	}
}

func TestVim_NormalMode_Navigation(t *testing.T) {
	t.Parallel()

	inp := NewInput()
	inp.SetFocused(true)
	inp.SetText("hello world")
	// SetText puts cursor at end (11)

	inp.SetVimEnabled(true)
	// Enabling vim puts us in Normal mode

	// h: move left
	inp.HandleInput("h")
	if inp.CursorPos() != 10 {
		t.Errorf("h: expected cursor at 10, got %d", inp.CursorPos())
	}

	// l: move right
	inp.HandleInput("l")
	if inp.CursorPos() != 11 {
		t.Errorf("l: expected cursor at 11, got %d", inp.CursorPos())
	}

	// 0: move to start
	inp.HandleInput("0")
	if inp.CursorPos() != 0 {
		t.Errorf("0: expected cursor at 0, got %d", inp.CursorPos())
	}

	// $: move to end
	inp.HandleInput("$")
	if inp.CursorPos() != 11 {
		t.Errorf("$: expected cursor at 11, got %d", inp.CursorPos())
	}

	// Left arrow also works in normal mode
	inp.HandleInput("\x1b[D") // left arrow
	if inp.CursorPos() != 10 {
		t.Errorf("left arrow: expected cursor at 10, got %d", inp.CursorPos())
	}

	// Right arrow also works in normal mode
	inp.HandleInput("\x1b[C") // right arrow
	if inp.CursorPos() != 11 {
		t.Errorf("right arrow: expected cursor at 11, got %d", inp.CursorPos())
	}
}

func TestVim_NormalMode_InsertTransitions(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		key        string
		setupPos   int
		wantMode   VimMode
		wantCursor int
	}{
		{
			name:       "i enters insert at cursor",
			key:        "i",
			setupPos:   5,
			wantMode:   VimInsert,
			wantCursor: 5,
		},
		{
			name:       "a enters insert after cursor",
			key:        "a",
			setupPos:   5,
			wantMode:   VimInsert,
			wantCursor: 6,
		},
		{
			name:       "A enters insert at end of line",
			key:        "A",
			setupPos:   3,
			wantMode:   VimInsert,
			wantCursor: 11, // len("hello world")
		},
		{
			name:       "I enters insert at start of line",
			key:        "I",
			setupPos:   7,
			wantMode:   VimInsert,
			wantCursor: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			inp := NewInput()
			inp.SetFocused(true)
			inp.SetText("hello world")
			inp.SetVimEnabled(true)

			// Move cursor to setup position
			inp.cursor = tt.setupPos

			inp.HandleInput(tt.key)

			if inp.VimMode() != tt.wantMode {
				t.Errorf("expected mode %d, got %d", tt.wantMode, inp.VimMode())
			}
			if inp.CursorPos() != tt.wantCursor {
				t.Errorf("expected cursor at %d, got %d", tt.wantCursor, inp.CursorPos())
			}
		})
	}
}

func TestVim_InsertMode_Escape(t *testing.T) {
	t.Parallel()

	inp := NewInput()
	inp.SetFocused(true)
	inp.SetText("hello")
	inp.SetVimEnabled(true)

	// Enter insert mode
	inp.HandleInput("i")
	if inp.VimMode() != VimInsert {
		t.Fatalf("expected insert mode after 'i', got %d", inp.VimMode())
	}

	// Escape returns to normal
	inp.HandleInput("\x1b")
	if inp.VimMode() != VimNormal {
		t.Errorf("expected normal mode after Esc, got %d", inp.VimMode())
	}
}

func TestVim_NormalMode_Delete(t *testing.T) {
	t.Parallel()

	t.Run("x deletes char at cursor", func(t *testing.T) {
		t.Parallel()

		inp := NewInput()
		inp.SetFocused(true)
		inp.SetText("hello")
		inp.SetVimEnabled(true)

		// cursor at end (5); move to position 0
		inp.HandleInput("0")
		inp.HandleInput("x")

		if inp.Text() != "ello" {
			t.Errorf("expected 'ello', got %q", inp.Text())
		}
		if inp.CursorPos() != 0 {
			t.Errorf("expected cursor at 0, got %d", inp.CursorPos())
		}
	})

	t.Run("dd clears entire line", func(t *testing.T) {
		t.Parallel()

		inp := NewInput()
		inp.SetFocused(true)
		inp.SetText("hello world")
		inp.SetVimEnabled(true)

		inp.HandleInput("d")
		inp.HandleInput("d")

		if inp.Text() != "" {
			t.Errorf("expected empty after dd, got %q", inp.Text())
		}
		if inp.CursorPos() != 0 {
			t.Errorf("expected cursor at 0, got %d", inp.CursorPos())
		}
	})

	t.Run("dw deletes to next word", func(t *testing.T) {
		t.Parallel()

		inp := NewInput()
		inp.SetFocused(true)
		inp.SetText("hello world")
		inp.SetVimEnabled(true)

		inp.HandleInput("0") // go to start
		inp.HandleInput("d")
		inp.HandleInput("w")

		if inp.Text() != "world" {
			t.Errorf("expected 'world' after dw, got %q", inp.Text())
		}
		if inp.CursorPos() != 0 {
			t.Errorf("expected cursor at 0, got %d", inp.CursorPos())
		}
	})
}

func TestVim_NormalMode_Change(t *testing.T) {
	t.Parallel()

	t.Run("cc clears line and enters insert mode", func(t *testing.T) {
		t.Parallel()

		inp := NewInput()
		inp.SetFocused(true)
		inp.SetText("hello world")
		inp.SetVimEnabled(true)

		inp.HandleInput("c")
		inp.HandleInput("c")

		if inp.Text() != "" {
			t.Errorf("expected empty after cc, got %q", inp.Text())
		}
		if inp.VimMode() != VimInsert {
			t.Errorf("expected insert mode after cc, got %d", inp.VimMode())
		}
	})

	t.Run("cw changes to next word and enters insert mode", func(t *testing.T) {
		t.Parallel()

		inp := NewInput()
		inp.SetFocused(true)
		inp.SetText("hello world")
		inp.SetVimEnabled(true)

		inp.HandleInput("0") // go to start
		inp.HandleInput("c")
		inp.HandleInput("w")

		if inp.Text() != "world" {
			t.Errorf("expected 'world' after cw, got %q", inp.Text())
		}
		if inp.VimMode() != VimInsert {
			t.Errorf("expected insert mode after cw, got %d", inp.VimMode())
		}
	})
}

func TestVim_NormalMode_WordMotion(t *testing.T) {
	t.Parallel()

	t.Run("w moves to next word start", func(t *testing.T) {
		t.Parallel()

		inp := NewInput()
		inp.SetFocused(true)
		inp.SetText("hello world foo")
		inp.SetVimEnabled(true)

		inp.HandleInput("0") // pos 0
		inp.HandleInput("w") // should go to 6 (start of "world")

		if inp.CursorPos() != 6 {
			t.Errorf("w: expected cursor at 6, got %d", inp.CursorPos())
		}

		inp.HandleInput("w") // should go to 12 (start of "foo")
		if inp.CursorPos() != 12 {
			t.Errorf("w: expected cursor at 12, got %d", inp.CursorPos())
		}
	})

	t.Run("b moves to previous word start", func(t *testing.T) {
		t.Parallel()

		inp := NewInput()
		inp.SetFocused(true)
		inp.SetText("hello world foo")
		inp.SetVimEnabled(true)

		// cursor at end (15)
		inp.HandleInput("b") // should go to 12 (start of "foo")
		if inp.CursorPos() != 12 {
			t.Errorf("b: expected cursor at 12, got %d", inp.CursorPos())
		}

		inp.HandleInput("b") // should go to 6 (start of "world")
		if inp.CursorPos() != 6 {
			t.Errorf("b: expected cursor at 6, got %d", inp.CursorPos())
		}

		inp.HandleInput("b") // should go to 0 (start of "hello")
		if inp.CursorPos() != 0 {
			t.Errorf("b: expected cursor at 0, got %d", inp.CursorPos())
		}
	})

	t.Run("e moves to end of word", func(t *testing.T) {
		t.Parallel()

		inp := NewInput()
		inp.SetFocused(true)
		inp.SetText("hello world foo")
		inp.SetVimEnabled(true)

		inp.HandleInput("0") // pos 0
		inp.HandleInput("e") // should go to 4 (end of "hello": 'o' at index 4)

		if inp.CursorPos() != 4 {
			t.Errorf("e: expected cursor at 4, got %d", inp.CursorPos())
		}

		inp.HandleInput("e") // should go to 10 (end of "world": 'd' at index 10)
		if inp.CursorPos() != 10 {
			t.Errorf("e: expected cursor at 10, got %d", inp.CursorPos())
		}
	})
}

func TestVim_InsertMode_TextEditing(t *testing.T) {
	t.Parallel()

	inp := NewInput()
	inp.SetFocused(true)
	inp.SetVimEnabled(true)

	// Start in normal mode; enter insert
	inp.HandleInput("i")
	if inp.VimMode() != VimInsert {
		t.Fatalf("expected insert mode, got %d", inp.VimMode())
	}

	// Type text normally
	inp.HandleInput("H")
	inp.HandleInput("i")

	if inp.Text() != "Hi" {
		t.Errorf("expected 'Hi', got %q", inp.Text())
	}
	if inp.CursorPos() != 2 {
		t.Errorf("expected cursor at 2, got %d", inp.CursorPos())
	}

	// Backspace works in insert mode
	inp.HandleInput("\x7f")
	if inp.Text() != "H" {
		t.Errorf("expected 'H' after backspace, got %q", inp.Text())
	}
}

func TestVim_NormalMode_EscapeIsNoop(t *testing.T) {
	t.Parallel()

	inp := NewInput()
	inp.SetFocused(true)
	inp.SetText("hello")
	inp.SetVimEnabled(true)

	pos := inp.CursorPos()
	inp.HandleInput("\x1b") // Esc in normal mode
	if inp.VimMode() != VimNormal {
		t.Errorf("expected normal mode, got %d", inp.VimMode())
	}
	if inp.CursorPos() != pos {
		t.Errorf("expected cursor unchanged at %d, got %d", pos, inp.CursorPos())
	}
}

func TestVim_VisualMode_EscapeReturnsNormal(t *testing.T) {
	t.Parallel()

	inp := NewInput()
	inp.SetFocused(true)
	inp.SetText("hello")
	inp.SetVimEnabled(true)

	// Manually set visual mode for testing (since we don't implement v command yet)
	inp.vimMode = VimVisual

	inp.HandleInput("\x1b") // Esc from visual
	if inp.VimMode() != VimNormal {
		t.Errorf("expected normal mode after Esc from visual, got %d", inp.VimMode())
	}
}

func TestVim_ExistingInputBehavior_Preserved(t *testing.T) {
	t.Parallel()

	// Ensure that when vim is disabled, everything works as before
	inp := NewInput()
	inp.SetFocused(true)
	inp.HandleInput("H")
	inp.HandleInput("i")

	if inp.Text() != "Hi" {
		t.Errorf("expected 'Hi' with vim disabled, got %q", inp.Text())
	}

	// Ctrl+A goes home
	inp.HandleInput("\x01")
	if inp.CursorPos() != 0 {
		t.Errorf("expected cursor at 0, got %d", inp.CursorPos())
	}

	// Ctrl+E goes end
	inp.HandleInput("\x05")
	if inp.CursorPos() != 2 {
		t.Errorf("expected cursor at 2, got %d", inp.CursorPos())
	}
}

func TestVim_NormalMode_Undo(t *testing.T) {
	t.Parallel()

	inp := NewInput()
	inp.SetFocused(true)
	inp.SetText("hello")
	inp.SetVimEnabled(true)

	// Delete char with x, then undo with u
	inp.HandleInput("0")
	inp.HandleInput("x") // deletes 'h', text = "ello"
	if inp.Text() != "ello" {
		t.Fatalf("expected 'ello' after x, got %q", inp.Text())
	}

	inp.HandleInput("u") // undo -> back to "hello"
	if inp.Text() != "hello" {
		t.Errorf("expected 'hello' after u, got %q", inp.Text())
	}
}

func TestVim_PendingOpCancelledByEscape(t *testing.T) {
	t.Parallel()

	inp := NewInput()
	inp.SetFocused(true)
	inp.SetText("hello world")
	inp.SetVimEnabled(true)

	// Press 'd' to start pending delete; then Esc to cancel
	inp.HandleInput("d")
	inp.HandleInput("\x1b")

	// Text should be unchanged
	if inp.Text() != "hello world" {
		t.Errorf("expected 'hello world' after d+Esc, got %q", inp.Text())
	}
}
