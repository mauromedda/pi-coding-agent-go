// ABOUTME: Defines the Key type and ParseKey for terminal keyboard input parsing.
// ABOUTME: Handles printable runes, control characters, and delegates escape sequences to legacy/kitty parsers.

package key

import (
	"fmt"
	"unicode/utf8"
)

// Key represents a parsed keyboard input event.
type Key struct {
	Type  KeyType
	Rune  rune // For printable characters
	Alt   bool
	Ctrl  bool
	Shift bool
}

// KeyType enumerates the kinds of key events the TUI can receive.
type KeyType int

const (
	KeyRune     KeyType = iota // Printable character
	KeyEnter                   // Enter / Return
	KeyTab                     // Tab
	KeyBackTab                 // Shift+Tab
	KeyBackspace               // Backspace / DEL (0x7F)
	KeyDelete                  // Delete key
	KeyUp                      // Arrow up
	KeyDown                    // Arrow down
	KeyLeft                    // Arrow left
	KeyRight                   // Arrow right
	KeyHome                    // Home
	KeyEnd                     // End
	KeyPageUp                  // Page Up
	KeyPageDown                // Page Down
	KeyEscape                  // Escape
	KeyCtrlC                   // Ctrl+C
	KeyCtrlD                   // Ctrl+D
	KeyCtrlG                   // Ctrl+G
	KeyCtrlL                   // Ctrl+L
	KeyCtrlO                   // Ctrl+O
	KeyCtrlR                   // Ctrl+R
	KeyUnknown                 // Unrecognized input
)

// ctrlKeys maps control byte values (0x01..0x1A) to their Key representations.
var ctrlKeys = map[byte]Key{
	0x03: {Type: KeyCtrlC, Ctrl: true},
	0x04: {Type: KeyCtrlD, Ctrl: true},
	0x07: {Type: KeyCtrlG, Ctrl: true},
	0x0c: {Type: KeyCtrlL, Ctrl: true},
	0x0f: {Type: KeyCtrlO, Ctrl: true},
	0x12: {Type: KeyCtrlR, Ctrl: true},
}

// ParseKey parses raw terminal input data into a Key.
// It handles single runes, control characters, and escape sequences.
func ParseKey(data string) Key {
	if len(data) == 0 {
		return Key{Type: KeyUnknown}
	}

	// Single-byte fast path
	if len(data) == 1 {
		return parseSingleByte(data[0])
	}

	// Escape sequence path
	if data[0] == 0x1b {
		return parseEscapeSequence(data)
	}

	// Multi-byte UTF-8 rune
	r, _ := utf8.DecodeRuneInString(data)
	if r == utf8.RuneError {
		return Key{Type: KeyUnknown}
	}
	return Key{Type: KeyRune, Rune: r}
}

// parseSingleByte handles a single-byte input (ASCII or control character).
func parseSingleByte(b byte) Key {
	switch {
	case b == 0x0d:
		return Key{Type: KeyEnter}
	case b == 0x09:
		return Key{Type: KeyTab}
	case b == 0x7f:
		return Key{Type: KeyBackspace}
	case b == 0x1b:
		return Key{Type: KeyEscape}
	case b >= 0x20 && b <= 0x7e:
		return Key{Type: KeyRune, Rune: rune(b)}
	}

	if k, ok := ctrlKeys[b]; ok {
		return k
	}
	return Key{Type: KeyUnknown}
}

// parseEscapeSequence delegates to legacy and kitty parsers for ESC-prefixed data.
func parseEscapeSequence(data string) Key {
	// Try Kitty protocol first (future-proofing)
	if k, ok := ParseKittyKey(data); ok {
		return k
	}

	// Try legacy escape sequences
	if k, ok := legacySequences[data]; ok {
		return k
	}

	// Lone ESC
	if len(data) == 1 {
		return Key{Type: KeyEscape}
	}

	// Alt+letter: ESC followed by a single printable byte (0x20..0x7e)
	if len(data) == 2 && data[1] >= 0x20 && data[1] <= 0x7e {
		return Key{Type: KeyRune, Rune: rune(data[1]), Alt: true}
	}

	return Key{Type: KeyUnknown}
}

// keyTypeNames provides human-readable labels for each KeyType.
var keyTypeNames = map[KeyType]string{
	KeyEnter:    "Enter",
	KeyTab:      "Tab",
	KeyBackTab:  "BackTab",
	KeyBackspace: "Backspace",
	KeyDelete:   "Delete",
	KeyUp:       "Up",
	KeyDown:     "Down",
	KeyLeft:     "Left",
	KeyRight:    "Right",
	KeyHome:     "Home",
	KeyEnd:      "End",
	KeyPageUp:   "PageUp",
	KeyPageDown: "PageDown",
	KeyEscape:   "Escape",
	KeyCtrlC:    "Ctrl+C",
	KeyCtrlD:    "Ctrl+D",
	KeyCtrlG:    "Ctrl+G",
	KeyCtrlL:    "Ctrl+L",
	KeyCtrlO:    "Ctrl+O",
	KeyCtrlR:    "Ctrl+R",
	KeyUnknown:  "Unknown",
}

// String returns a human-readable representation of the Key for debug display.
func (k Key) String() string {
	if k.Type == KeyRune {
		return formatRuneKey(k)
	}
	if name, ok := keyTypeNames[k.Type]; ok {
		return name
	}
	return "Unknown"
}

// formatRuneKey builds a display string for printable rune keys with modifiers.
func formatRuneKey(k Key) string {
	s := string(k.Rune)
	if k.Alt {
		s = fmt.Sprintf("Alt+%s", s)
	}
	return s
}
