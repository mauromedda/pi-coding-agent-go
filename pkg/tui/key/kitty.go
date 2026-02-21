// ABOUTME: Kitty keyboard protocol CSI u sequence parser for enhanced key input.
// ABOUTME: Parses unicode codepoints, modifier bitmasks, event types, and alternate keys per the Kitty spec.

package key

import "strconv"

// Kitty modifier bitmask values (encoded as modifiers-1 in the wire format).
const (
	kittyShift = 1 << iota // bit 0
	kittyAlt               // bit 1
	kittyCtrl              // bit 2
)

// ctrlKeyTypes maps lowercase rune codepoints to their Ctrl+<key> types.
var ctrlKeyTypes = map[rune]KeyType{
	'c': KeyCtrlC,
	'd': KeyCtrlD,
	'g': KeyCtrlG,
	'l': KeyCtrlL,
	'o': KeyCtrlO,
	'r': KeyCtrlR,
}

// tildeKeyTypes maps CSI number~ codes to their key types.
var tildeKeyTypes = map[int]KeyType{
	3: KeyDelete,
	5: KeyPageUp,
	6: KeyPageDown,
}

// letterKeyTypes maps CSI letter terminators to their key types.
var letterKeyTypes = map[byte]KeyType{
	'A': KeyUp,
	'B': KeyDown,
	'C': KeyRight,
	'D': KeyLeft,
	'H': KeyHome,
	'F': KeyEnd,
}

// ParseKittyKey parses a Kitty protocol escape sequence from raw terminal input.
// It handles three formats:
//   - CSI <codepoint>[:<shifted>] [; <modifiers>[:<event>]] u
//   - CSI <number> ; <modifiers> ~      (functional keys)
//   - CSI 1 ; <modifiers> <letter>      (arrow/nav keys with modifiers)
//
// Returns the parsed Key and true on success; zero Key and false otherwise.
func ParseKittyKey(data string) (Key, bool) {
	if len(data) < 4 || data[0] != 0x1b || data[1] != '[' {
		return Key{}, false
	}

	body := data[2:]
	terminator := body[len(body)-1]

	switch terminator {
	case 'u':
		return parseCSIu(body[:len(body)-1])
	case '~':
		return parseTilde(body[:len(body)-1])
	case 'A', 'B', 'C', 'D', 'H', 'F':
		return parseLetterTerminator(body[:len(body)-1], terminator)
	default:
		return Key{}, false
	}
}

// parseCSIu handles the CSI <codepoint>[:<shifted>] [; <modifiers>[:<event>]] u format.
func parseCSIu(body string) (Key, bool) {
	codepointStr, modifierStr := splitOnSemicolon(body)

	codepoint, err := parseCodepoint(codepointStr)
	if err != nil {
		return Key{}, false
	}

	mods, event, err := parseModifiers(modifierStr)
	if err != nil {
		return Key{}, false
	}

	// Event type 3 = key release; ignore it
	if event == 3 {
		return Key{}, false
	}

	return buildKey(codepoint, mods), true
}

// parseTilde handles the CSI <number> ; <modifiers> ~ format for functional keys.
func parseTilde(body string) (Key, bool) {
	numStr, modifierStr := splitOnSemicolon(body)

	num, err := strconv.Atoi(numStr)
	if err != nil {
		return Key{}, false
	}

	kt, ok := tildeKeyTypes[num]
	if !ok {
		return Key{}, false
	}

	mods, _, err := parseModifiers(modifierStr)
	if err != nil {
		return Key{}, false
	}

	k := Key{Type: kt}
	applyModifiers(&k, mods)
	return k, true
}

// parseLetterTerminator handles CSI 1 ; <modifiers> <letter> for arrow/nav keys.
func parseLetterTerminator(body string, letter byte) (Key, bool) {
	kt, ok := letterKeyTypes[letter]
	if !ok {
		return Key{}, false
	}

	_, modifierStr := splitOnSemicolon(body)
	if modifierStr == "" {
		return Key{}, false
	}

	mods, _, err := parseModifiers(modifierStr)
	if err != nil {
		return Key{}, false
	}

	k := Key{Type: kt}
	applyModifiers(&k, mods)
	return k, true
}

// splitOnSemicolon splits a string into at most two parts on the first ';'.
func splitOnSemicolon(s string) (string, string) {
	for i := 0; i < len(s); i++ {
		if s[i] == ';' {
			return s[:i], s[i+1:]
		}
	}
	return s, ""
}

// splitOnColon splits a string into at most two parts on the first ':'.
func splitOnColon(s string) (string, string) {
	for i := 0; i < len(s); i++ {
		if s[i] == ':' {
			return s[:i], s[i+1:]
		}
	}
	return s, ""
}

// parseCodepoint extracts the primary unicode codepoint from a potentially colon-delimited string.
// Format: <codepoint>[:<shifted_key>[:<base_key>]]
func parseCodepoint(s string) (rune, error) {
	primary, _ := splitOnColon(s)
	n, err := strconv.Atoi(primary)
	if err != nil {
		return 0, err
	}
	return rune(n), nil
}

// parseModifiers parses the modifier and optional event type from a string.
// Format: <modifiers>[:<event_type>]
// Returns the decoded bitmask, event type (0 if absent), and any error.
func parseModifiers(s string) (int, int, error) {
	if s == "" {
		return 0, 0, nil
	}

	modStr, eventStr := splitOnColon(s)

	modVal, err := strconv.Atoi(modStr)
	if err != nil {
		return 0, 0, err
	}
	// Wire format is modifiers-1; decode the bitmask
	mods := modVal - 1

	event := 0
	if eventStr != "" {
		event, err = strconv.Atoi(eventStr)
		if err != nil {
			return 0, 0, err
		}
	}

	return mods, event, nil
}

// buildKey constructs a Key from a unicode codepoint and modifier bitmask.
func buildKey(codepoint rune, mods int) Key {
	k := mapCodepointToKey(codepoint)

	// For Ctrl+<letter>, check if there is a specific KeyType
	if mods&kittyCtrl != 0 {
		if kt, ok := ctrlKeyTypes[codepoint]; ok {
			k = Key{Type: kt}
		}
	}

	// Tab + Shift = BackTab
	if k.Type == KeyTab && mods&kittyShift != 0 {
		k = Key{Type: KeyBackTab}
	}

	applyModifiers(&k, mods)
	return k
}

// mapCodepointToKey converts a unicode codepoint to a base Key without modifiers.
func mapCodepointToKey(cp rune) Key {
	switch cp {
	case 13:
		return Key{Type: KeyEnter}
	case 9:
		return Key{Type: KeyTab}
	case 127:
		return Key{Type: KeyBackspace}
	case 27:
		return Key{Type: KeyEscape}
	default:
		return Key{Type: KeyRune, Rune: cp}
	}
}

// applyModifiers sets the modifier flags on a Key from the decoded bitmask.
func applyModifiers(k *Key, mods int) {
	if mods&kittyShift != 0 {
		k.Shift = true
	}
	if mods&kittyAlt != 0 {
		k.Alt = true
	}
	if mods&kittyCtrl != 0 {
		k.Ctrl = true
	}
}
