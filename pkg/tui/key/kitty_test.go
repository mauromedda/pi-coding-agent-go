// ABOUTME: Tests for Kitty keyboard protocol CSI u sequence parsing.
// ABOUTME: Covers unicode codepoints, modifier combinations, special keys, and edge cases.

package key

import "testing"

func TestParseKittyKey(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		data    string
		want    Key
		wantOK  bool
	}{
		// Basic CSI u sequences: ESC [ <codepoint> u
		{
			name:   "lowercase a",
			data:   "\x1b[97u",
			want:   Key{Type: KeyRune, Rune: 'a'},
			wantOK: true,
		},
		{
			name:   "uppercase A",
			data:   "\x1b[65u",
			want:   Key{Type: KeyRune, Rune: 'A'},
			wantOK: true,
		},
		{
			name:   "space",
			data:   "\x1b[32u",
			want:   Key{Type: KeyRune, Rune: ' '},
			wantOK: true,
		},
		{
			name:   "digit 1",
			data:   "\x1b[49u",
			want:   Key{Type: KeyRune, Rune: '1'},
			wantOK: true,
		},

		// Modifiers: ESC [ <codepoint> ; <modifiers> u
		// Modifier encoding: value = 1 + bitmask (shift=1, alt=2, ctrl=4)
		{
			name:   "shift+a",
			data:   "\x1b[97;2u",
			want:   Key{Type: KeyRune, Rune: 'a', Shift: true},
			wantOK: true,
		},
		{
			name:   "alt+a",
			data:   "\x1b[97;3u",
			want:   Key{Type: KeyRune, Rune: 'a', Alt: true},
			wantOK: true,
		},
		{
			name:   "ctrl+a",
			data:   "\x1b[97;5u",
			want:   Key{Type: KeyRune, Rune: 'a', Ctrl: true},
			wantOK: true,
		},
		{
			name:   "ctrl+shift+a",
			data:   "\x1b[97;6u",
			want:   Key{Type: KeyRune, Rune: 'a', Ctrl: true, Shift: true},
			wantOK: true,
		},
		{
			name:   "ctrl+alt+a",
			data:   "\x1b[97;7u",
			want:   Key{Type: KeyRune, Rune: 'a', Ctrl: true, Alt: true},
			wantOK: true,
		},
		{
			name:   "ctrl+alt+shift+a",
			data:   "\x1b[97;8u",
			want:   Key{Type: KeyRune, Rune: 'a', Ctrl: true, Alt: true, Shift: true},
			wantOK: true,
		},
		{
			name:   "no modifiers explicit (modifier=1)",
			data:   "\x1b[97;1u",
			want:   Key{Type: KeyRune, Rune: 'a'},
			wantOK: true,
		},

		// Special keys encoded as CSI u
		{
			name:   "enter",
			data:   "\x1b[13u",
			want:   Key{Type: KeyEnter},
			wantOK: true,
		},
		{
			name:   "tab",
			data:   "\x1b[9u",
			want:   Key{Type: KeyTab},
			wantOK: true,
		},
		{
			name:   "backspace",
			data:   "\x1b[127u",
			want:   Key{Type: KeyBackspace},
			wantOK: true,
		},
		{
			name:   "escape key",
			data:   "\x1b[27u",
			want:   Key{Type: KeyEscape},
			wantOK: true,
		},
		{
			name:   "shift+tab (backtab)",
			data:   "\x1b[9;2u",
			want:   Key{Type: KeyBackTab, Shift: true},
			wantOK: true,
		},
		{
			name:   "ctrl+c via kitty",
			data:   "\x1b[99;5u",
			want:   Key{Type: KeyCtrlC, Ctrl: true},
			wantOK: true,
		},
		{
			name:   "ctrl+d via kitty",
			data:   "\x1b[100;5u",
			want:   Key{Type: KeyCtrlD, Ctrl: true},
			wantOK: true,
		},
		{
			name:   "ctrl+g via kitty",
			data:   "\x1b[103;5u",
			want:   Key{Type: KeyCtrlG, Ctrl: true},
			wantOK: true,
		},
		{
			name:   "ctrl+l via kitty",
			data:   "\x1b[108;5u",
			want:   Key{Type: KeyCtrlL, Ctrl: true},
			wantOK: true,
		},
		{
			name:   "ctrl+o via kitty",
			data:   "\x1b[111;5u",
			want:   Key{Type: KeyCtrlO, Ctrl: true},
			wantOK: true,
		},
		{
			name:   "ctrl+r via kitty",
			data:   "\x1b[114;5u",
			want:   Key{Type: KeyCtrlR, Ctrl: true},
			wantOK: true,
		},

		// Functional keys with CSI number ~ format: ESC [ number ; modifiers ~
		{
			name:   "delete key",
			data:   "\x1b[3;1~",
			want:   Key{Type: KeyDelete},
			wantOK: true,
		},
		{
			name:   "page up",
			data:   "\x1b[5;1~",
			want:   Key{Type: KeyPageUp},
			wantOK: true,
		},
		{
			name:   "page down",
			data:   "\x1b[6;1~",
			want:   Key{Type: KeyPageDown},
			wantOK: true,
		},
		{
			name:   "shift+delete",
			data:   "\x1b[3;2~",
			want:   Key{Type: KeyDelete, Shift: true},
			wantOK: true,
		},

		// Arrow keys with modifiers: ESC [ 1 ; modifiers <A-D>
		{
			name:   "shift+up",
			data:   "\x1b[1;2A",
			want:   Key{Type: KeyUp, Shift: true},
			wantOK: true,
		},
		{
			name:   "alt+down",
			data:   "\x1b[1;3B",
			want:   Key{Type: KeyDown, Alt: true},
			wantOK: true,
		},
		{
			name:   "ctrl+right",
			data:   "\x1b[1;5C",
			want:   Key{Type: KeyRight, Ctrl: true},
			wantOK: true,
		},
		{
			name:   "ctrl+shift+left",
			data:   "\x1b[1;6D",
			want:   Key{Type: KeyLeft, Ctrl: true, Shift: true},
			wantOK: true,
		},
		{
			name:   "shift+home",
			data:   "\x1b[1;2H",
			want:   Key{Type: KeyHome, Shift: true},
			wantOK: true,
		},
		{
			name:   "ctrl+end",
			data:   "\x1b[1;5F",
			want:   Key{Type: KeyEnd, Ctrl: true},
			wantOK: true,
		},

		// Extended Kitty protocol with event type: ESC [ codepoint ; modifiers:event u
		{
			name:   "key press event (event=1)",
			data:   "\x1b[97;1:1u",
			want:   Key{Type: KeyRune, Rune: 'a'},
			wantOK: true,
		},
		{
			name:   "key repeat event (event=2)",
			data:   "\x1b[97;1:2u",
			want:   Key{Type: KeyRune, Rune: 'a'},
			wantOK: true,
		},
		{
			name:   "key release event (event=3) ignored",
			data:   "\x1b[97;1:3u",
			want:   Key{},
			wantOK: false,
		},
		{
			name:   "shift+b with press event",
			data:   "\x1b[98;2:1u",
			want:   Key{Type: KeyRune, Rune: 'b', Shift: true},
			wantOK: true,
		},

		// Alternate key encoding: ESC [ codepoint:shifted_key ; modifiers u
		{
			name:   "a with shifted A alternate key",
			data:   "\x1b[97:65u",
			want:   Key{Type: KeyRune, Rune: 'a'},
			wantOK: true,
		},
		{
			name:   "a with shifted A and modifiers",
			data:   "\x1b[97:65;2u",
			want:   Key{Type: KeyRune, Rune: 'a', Shift: true},
			wantOK: true,
		},

		// Unicode characters
		{
			name:   "unicode e-acute",
			data:   "\x1b[233u",
			want:   Key{Type: KeyRune, Rune: '\u00e9'},
			wantOK: true,
		},
		{
			name:   "unicode CJK character",
			data:   "\x1b[20013u",
			want:   Key{Type: KeyRune, Rune: '\u4e2d'},
			wantOK: true,
		},

		// Invalid inputs
		{
			name:   "empty string",
			data:   "",
			want:   Key{},
			wantOK: false,
		},
		{
			name:   "no ESC prefix",
			data:   "[97u",
			want:   Key{},
			wantOK: false,
		},
		{
			name:   "missing CSI bracket",
			data:   "\x1b97u",
			want:   Key{},
			wantOK: false,
		},
		{
			name:   "not a CSI u sequence",
			data:   "\x1b[A",
			want:   Key{},
			wantOK: false,
		},
		{
			name:   "invalid codepoint",
			data:   "\x1b[abcu",
			want:   Key{},
			wantOK: false,
		},
		{
			name:   "missing terminator",
			data:   "\x1b[97",
			want:   Key{},
			wantOK: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got, ok := ParseKittyKey(tt.data)
			if ok != tt.wantOK {
				t.Fatalf("ParseKittyKey(%q) ok = %v, want %v", tt.data, ok, tt.wantOK)
			}
			if !tt.wantOK {
				return
			}
			if got.Type != tt.want.Type {
				t.Errorf("ParseKittyKey(%q).Type = %v, want %v", tt.data, got.Type, tt.want.Type)
			}
			if got.Rune != tt.want.Rune {
				t.Errorf("ParseKittyKey(%q).Rune = %q, want %q", tt.data, got.Rune, tt.want.Rune)
			}
			if got.Ctrl != tt.want.Ctrl {
				t.Errorf("ParseKittyKey(%q).Ctrl = %v, want %v", tt.data, got.Ctrl, tt.want.Ctrl)
			}
			if got.Alt != tt.want.Alt {
				t.Errorf("ParseKittyKey(%q).Alt = %v, want %v", tt.data, got.Alt, tt.want.Alt)
			}
			if got.Shift != tt.want.Shift {
				t.Errorf("ParseKittyKey(%q).Shift = %v, want %v", tt.data, got.Shift, tt.want.Shift)
			}
		})
	}
}

func TestParseKittyKey_IntegrationWithParseKey(t *testing.T) {
	t.Parallel()

	// Verify that ParseKey delegates to ParseKittyKey for CSI u sequences
	tests := []struct {
		name string
		data string
		want Key
	}{
		{
			name: "kitty a via ParseKey",
			data: "\x1b[97u",
			want: Key{Type: KeyRune, Rune: 'a'},
		},
		{
			name: "kitty ctrl+c via ParseKey",
			data: "\x1b[99;5u",
			want: Key{Type: KeyCtrlC, Ctrl: true},
		},
		{
			name: "kitty enter via ParseKey",
			data: "\x1b[13u",
			want: Key{Type: KeyEnter},
		},
		{
			name: "kitty shift+up via ParseKey",
			data: "\x1b[1;2A",
			want: Key{Type: KeyUp, Shift: true},
		},
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
			if got.Alt != tt.want.Alt {
				t.Errorf("ParseKey(%q).Alt = %v, want %v", tt.data, got.Alt, tt.want.Alt)
			}
			if got.Shift != tt.want.Shift {
				t.Errorf("ParseKey(%q).Shift = %v, want %v", tt.data, got.Shift, tt.want.Shift)
			}
		})
	}
}
