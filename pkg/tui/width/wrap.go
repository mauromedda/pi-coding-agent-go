// ABOUTME: ANSI-aware text wrapping and truncation
// ABOUTME: WrapTextWithAnsi wraps at column boundaries; TruncateToWidth adds ellipsis

package width

import (
	"strings"

	"github.com/rivo/uniseg"
)

// WrapTextWithAnsi wraps s into lines of at most maxWidth visible columns.
// ANSI escape sequences are preserved and do not count toward width.
// Words are broken if they exceed maxWidth.
func WrapTextWithAnsi(s string, maxWidth int) []string {
	if maxWidth <= 0 {
		return nil
	}
	if s == "" {
		return []string{""}
	}

	var lines []string
	var currentLine strings.Builder
	currentWidth := 0
	var sgr ActiveSGR

	i := 0
	for i < len(s) {
		if s[i] == '\n' {
			lines = append(lines, currentLine.String())
			currentLine.Reset()
			currentWidth = 0
			// Carry SGR state to next line
			prefix := sgr.String()
			if prefix != "" {
				currentLine.WriteString(prefix)
			}
			i++
			continue
		}

		if s[i] == '\x1b' {
			end := skipANSISequence(s, i)
			seq := s[i:end]
			sgr.Apply(seq)
			currentLine.WriteString(seq)
			i = end
			continue
		}

		// Read a grapheme cluster
		cluster, rest, _, _ := uniseg.FirstGraphemeClusterInString(s[i:], -1)
		w := graphemeWidth(cluster)

		if currentWidth+w > maxWidth {
			lines = append(lines, currentLine.String())
			currentLine.Reset()
			currentWidth = 0
			prefix := sgr.String()
			if prefix != "" {
				currentLine.WriteString(prefix)
			}
		}

		currentLine.WriteString(cluster)
		currentWidth += w
		i += len(s[i:]) - len(rest)
	}

	lines = append(lines, currentLine.String())
	return lines
}

// TruncateToWidth truncates s to at most maxWidth visible columns.
// If truncation occurs, the last visible character is replaced with ellipsis.
func TruncateToWidth(s string, maxWidth int) string {
	if maxWidth <= 0 {
		return ""
	}
	w := VisibleWidth(s)
	if w <= maxWidth {
		return s
	}
	if maxWidth == 1 {
		return "\u2026" // single ellipsis character
	}

	var b strings.Builder
	col := 0
	target := maxWidth - 1 // Leave room for ellipsis
	i := 0
	for i < len(s) && col < target {
		if s[i] == '\x1b' {
			end := skipANSISequence(s, i)
			b.WriteString(s[i:end])
			i = end
			continue
		}
		cluster, rest, _, _ := uniseg.FirstGraphemeClusterInString(s[i:], -1)
		cw := graphemeWidth(cluster)
		if col+cw > target {
			break
		}
		b.WriteString(cluster)
		col += cw
		i += len(s[i:]) - len(rest)
	}
	b.WriteString("\x1b[0m") // Reset before ellipsis
	b.WriteRune('\u2026')
	return b.String()
}
