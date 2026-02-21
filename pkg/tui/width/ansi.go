// ABOUTME: ANSI escape sequence extraction, stripping, and SGR state tracking
// ABOUTME: Handles CSI sequences, OSC sequences, and basic ESC sequences

package width

import "strings"

// StripANSI removes all ANSI escape sequences from s.
func StripANSI(s string) string {
	if !containsESC(s) {
		return s
	}
	var b strings.Builder
	b.Grow(len(s))
	i := 0
	for i < len(s) {
		if s[i] == '\x1b' {
			end := skipANSISequence(s, i)
			i = end
			continue
		}
		b.WriteByte(s[i])
		i++
	}
	return b.String()
}

// ExtractANSI returns all ANSI escape sequences found in s, in order.
func ExtractANSI(s string) []string {
	if !containsESC(s) {
		return nil
	}
	var seqs []string
	i := 0
	for i < len(s) {
		if s[i] == '\x1b' {
			end := skipANSISequence(s, i)
			seqs = append(seqs, s[i:end])
			i = end
			continue
		}
		i++
	}
	return seqs
}

// containsESC is a fast check for the presence of ESC (0x1B).
func containsESC(s string) bool {
	return strings.ContainsRune(s, '\x1b')
}

// skipANSISequence advances past an ANSI escape sequence starting at s[i].
// Returns the index of the first byte after the sequence.
func skipANSISequence(s string, i int) int {
	if i >= len(s) || s[i] != '\x1b' {
		return i
	}
	i++ // skip ESC
	if i >= len(s) {
		return i
	}

	switch s[i] {
	case '[':
		// CSI sequence: ESC [ ... <final byte 0x40-0x7E>
		i++
		for i < len(s) {
			b := s[i]
			if b >= 0x40 && b <= 0x7E {
				return i + 1
			}
			i++
		}
		return i
	case ']':
		// OSC sequence: ESC ] ... (ST or BEL)
		i++
		for i < len(s) {
			if s[i] == '\x07' {
				return i + 1
			}
			if s[i] == '\x1b' && i+1 < len(s) && s[i+1] == '\\' {
				return i + 2
			}
			i++
		}
		return i
	case '(':
		// Designate character set: ESC ( <char>
		if i+1 < len(s) {
			return i + 2
		}
		return i + 1
	case '_', 'P', '^':
		// APC, DCS, PM: terminated by ST
		i++
		for i < len(s) {
			if s[i] == '\x1b' && i+1 < len(s) && s[i+1] == '\\' {
				return i + 2
			}
			i++
		}
		return i
	default:
		// Simple two-byte ESC sequence
		return i + 1
	}
}

// ActiveSGR tracks the current SGR (Select Graphic Rendition) state.
type ActiveSGR struct {
	codes []string
}

// Reset clears all SGR state.
func (a *ActiveSGR) Reset() {
	a.codes = a.codes[:0]
}

// Apply processes an SGR sequence and updates state.
func (a *ActiveSGR) Apply(seq string) {
	if seq == "\x1b[0m" || seq == "\x1b[m" {
		a.Reset()
		return
	}
	a.codes = append(a.codes, seq)
}

// String returns the combined SGR sequence to restore current state.
func (a *ActiveSGR) String() string {
	if len(a.codes) == 0 {
		return ""
	}
	var b strings.Builder
	for _, c := range a.codes {
		b.WriteString(c)
	}
	return b.String()
}
