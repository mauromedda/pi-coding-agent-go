// ABOUTME: Path normalization utilities for Unicode-aware file resolution
// ABOUTME: Handles NFD/NFC, curly quotes, narrow no-break spaces, tilde expansion

package tools

import (
	"os"
	"path/filepath"
	"strings"

	"golang.org/x/text/unicode/norm"
)

// NormalizeSpaces replaces Unicode space characters with ASCII space (U+0020).
// Covered codepoints: U+00A0, U+2000-U+200A, U+202F, U+205F, U+3000.
func NormalizeSpaces(s string) string {
	var b strings.Builder
	b.Grow(len(s))
	for _, r := range s {
		if isUnicodeSpace(r) {
			b.WriteByte(' ')
		} else {
			b.WriteRune(r)
		}
	}
	return b.String()
}

// isUnicodeSpace reports whether r is a non-ASCII Unicode space character.
func isUnicodeSpace(r rune) bool {
	switch {
	case r == '\u00A0': // no-break space
		return true
	case r >= '\u2000' && r <= '\u200A': // en/em/thin/hair/etc. spaces
		return true
	case r == '\u202F': // narrow no-break space
		return true
	case r == '\u205F': // medium mathematical space
		return true
	case r == '\u3000': // ideographic space
		return true
	}
	return false
}

// ExpandPath strips a leading "@" prefix, expands "~" to the user home
// directory, and normalizes Unicode spaces.
func ExpandPath(path string) string {
	// Strip leading "@" that LLMs sometimes prepend.
	path = strings.TrimPrefix(path, "@")

	// Expand leading "~" to home directory.
	if strings.HasPrefix(path, "~") {
		home, err := os.UserHomeDir()
		if err == nil {
			path = home + path[1:]
		}
	}

	return NormalizeSpaces(path)
}

// ResolveToCwd expands the path and, if it is relative, joins it with cwd.
// The result is always filepath.Clean'd.
func ResolveToCwd(path, cwd string) string {
	path = ExpandPath(path)
	if !filepath.IsAbs(path) {
		path = filepath.Join(cwd, path)
	}
	return filepath.Clean(path)
}

// ResolveReadPath tries up to five Unicode-variant resolutions of path against
// the filesystem. It returns the first variant whose os.Stat succeeds; if none
// match, it returns the direct resolution.
func ResolveReadPath(path, cwd string) string {
	candidates := []string{
		// 1. direct
		ResolveToCwd(path, cwd),
		// 2. narrow no-break space → ASCII space
		ResolveToCwd(NormalizeSpaces(strings.ReplaceAll(path, "\u202f", " ")), cwd),
		// 3. NFD normalization
		ResolveToCwd(norm.NFD.String(path), cwd),
		// 4. curly right single quote → straight apostrophe
		ResolveToCwd(strings.ReplaceAll(path, "\u2019", "'"), cwd),
		// 5. NFD + curly quote
		ResolveToCwd(norm.NFD.String(strings.ReplaceAll(path, "\u2019", "'")), cwd),
	}

	for _, c := range candidates {
		if _, err := os.Stat(c); err == nil {
			return c
		}
	}

	// Fallback: return the direct resolution even though it doesn't exist.
	return candidates[0]
}
