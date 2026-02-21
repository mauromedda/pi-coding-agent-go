// ABOUTME: Table-driven tests for Key parsing covering ASCII, control chars, and escape sequences.
// ABOUTME: Validates ParseKey against single runes, Ctrl combos, arrows, and unknown inputs.

package key

import "testing"

func TestParseKey(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		data string
		want Key
	}{
		// Single printable ASCII characters
		{name: "lowercase a", data: "a", want: Key{Type: KeyRune, Rune: 'a'}},
		{name: "lowercase z", data: "z", want: Key{Type: KeyRune, Rune: 'z'}},
		{name: "uppercase A", data: "A", want: Key{Type: KeyRune, Rune: 'A'}},
		{name: "digit 0", data: "0", want: Key{Type: KeyRune, Rune: '0'}},
		{name: "space", data: " ", want: Key{Type: KeyRune, Rune: ' '}},
		{name: "tilde", data: "~", want: Key{Type: KeyRune, Rune: '~'}},

		// Control characters
		{name: "ctrl+c", data: "\x03", want: Key{Type: KeyCtrlC, Ctrl: true}},
		{name: "ctrl+d", data: "\x04", want: Key{Type: KeyCtrlD, Ctrl: true}},
		{name: "ctrl+g", data: "\x07", want: Key{Type: KeyCtrlG, Ctrl: true}},
		{name: "ctrl+l", data: "\x0c", want: Key{Type: KeyCtrlL, Ctrl: true}},
		{name: "ctrl+o", data: "\x0f", want: Key{Type: KeyCtrlO, Ctrl: true}},
		{name: "ctrl+r", data: "\x12", want: Key{Type: KeyCtrlR, Ctrl: true}},

		// Enter, Tab, Backspace
		{name: "enter", data: "\r", want: Key{Type: KeyEnter}},
		{name: "tab", data: "\t", want: Key{Type: KeyTab}},
		{name: "backspace", data: "\x7f", want: Key{Type: KeyBackspace}},

		// Escape alone
		{name: "escape", data: "\x1b", want: Key{Type: KeyEscape}},

		// CSI arrow keys
		{name: "arrow up", data: "\x1b[A", want: Key{Type: KeyUp}},
		{name: "arrow down", data: "\x1b[B", want: Key{Type: KeyDown}},
		{name: "arrow right", data: "\x1b[C", want: Key{Type: KeyRight}},
		{name: "arrow left", data: "\x1b[D", want: Key{Type: KeyLeft}},

		// Home, End
		{name: "home", data: "\x1b[H", want: Key{Type: KeyHome}},
		{name: "end", data: "\x1b[F", want: Key{Type: KeyEnd}},

		// Page Up, Page Down, Delete
		{name: "page up", data: "\x1b[5~", want: Key{Type: KeyPageUp}},
		{name: "page down", data: "\x1b[6~", want: Key{Type: KeyPageDown}},
		{name: "delete", data: "\x1b[3~", want: Key{Type: KeyDelete}},

		// BackTab (Shift+Tab)
		{name: "backtab", data: "\x1b[Z", want: Key{Type: KeyBackTab, Shift: true}},

		// SS3 arrow keys
		{name: "SS3 up", data: "\x1bOA", want: Key{Type: KeyUp}},
		{name: "SS3 down", data: "\x1bOB", want: Key{Type: KeyDown}},
		{name: "SS3 right", data: "\x1bOC", want: Key{Type: KeyRight}},
		{name: "SS3 left", data: "\x1bOD", want: Key{Type: KeyLeft}},
		{name: "SS3 home", data: "\x1bOH", want: Key{Type: KeyHome}},
		{name: "SS3 end", data: "\x1bOF", want: Key{Type: KeyEnd}},

		// Unknown escape sequence
		{name: "unknown escape", data: "\x1b[99Z", want: Key{Type: KeyUnknown}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got := ParseKey(tt.data)
			if got.Type != tt.want.Type {
				t.Errorf("ParseKey(%q).Type = %v, want %v", tt.data, got.Type, tt.want.Type)
			}
			if got.Rune != tt.want.Rune {
				t.Errorf("ParseKey(%q).Rune = %q, want %q", tt.data, got.Rune, tt.want.Rune)
			}
			if got.Ctrl != tt.want.Ctrl {
				t.Errorf("ParseKey(%q).Ctrl = %v, want %v", tt.data, got.Ctrl, tt.want.Ctrl)
			}
			if got.Shift != tt.want.Shift {
				t.Errorf("ParseKey(%q).Shift = %v, want %v", tt.data, got.Shift, tt.want.Shift)
			}
		})
	}
}

func TestKeyString(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		key  Key
		want string
	}{
		{name: "rune a", key: Key{Type: KeyRune, Rune: 'a'}, want: "a"},
		{name: "enter", key: Key{Type: KeyEnter}, want: "Enter"},
		{name: "ctrl+c", key: Key{Type: KeyCtrlC, Ctrl: true}, want: "Ctrl+C"},
		{name: "arrow up", key: Key{Type: KeyUp}, want: "Up"},
		{name: "unknown", key: Key{Type: KeyUnknown}, want: "Unknown"},
		{name: "alt rune", key: Key{Type: KeyRune, Rune: 'x', Alt: true}, want: "Alt+x"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got := tt.key.String()
			if got != tt.want {
				t.Errorf("Key.String() = %q, want %q", got, tt.want)
			}
		})
	}
}
