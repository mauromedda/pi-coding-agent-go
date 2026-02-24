// ABOUTME: Partial JSON parser for streaming tool call arguments
// ABOUTME: Completes truncated JSON strings/objects/arrays, returns best-effort map

package partjson

import (
	"encoding/json"
	"strings"
)

// Parse attempts to parse potentially incomplete JSON into a map.
// If standard parsing fails, it tries to complete the JSON by closing
// open strings, arrays, and objects. Returns an empty map on total failure.
func Parse(s string) map[string]any {
	if s == "" {
		return map[string]any{}
	}

	// Fast path: try standard parse first
	var result map[string]any
	if err := json.Unmarshal([]byte(s), &result); err == nil {
		return result
	}

	// Slow path: attempt to complete the JSON
	completed := completeJSON(s)
	if err := json.Unmarshal([]byte(completed), &result); err == nil {
		return result
	}

	return map[string]any{}
}

// completeJSON attempts to close any open JSON structures.
func completeJSON(s string) string {
	var closers []byte
	inString := false
	escaped := false

	for i := 0; i < len(s); i++ {
		c := s[i]

		if escaped {
			escaped = false
			continue
		}

		if c == '\\' && inString {
			escaped = true
			continue
		}

		if c == '"' {
			if inString {
				inString = false
			} else {
				inString = true
			}
			continue
		}

		if inString {
			continue
		}

		switch c {
		case '{':
			closers = append(closers, '}')
		case '[':
			closers = append(closers, ']')
		case '}', ']':
			if len(closers) > 0 {
				closers = closers[:len(closers)-1]
			}
		}
	}

	result := s

	// Close open string. If the last character was a backslash (escaped is true),
	// appending just `"` would produce `\"` which escapes the closing quote.
	// Add an extra backslash to properly terminate the escape sequence.
	if inString {
		if escaped {
			result += `\`
		}
		result += `"`
	}

	// Remove trailing comma or colon that would make JSON invalid
	trimmed := trimTrailingJunk(result)

	// Close open structures in reverse order
	var sb strings.Builder
	sb.WriteString(trimmed)
	for i := len(closers) - 1; i >= 0; i-- {
		sb.WriteByte(closers[i])
	}

	return sb.String()
}

// trimTrailingJunk removes trailing characters that would invalidate JSON
// after we close an open string (trailing comma, colon, incomplete key-value).
// Also strips incomplete boolean/null literals (e.g. "tr", "fal", "nu") that
// appear when JSON is truncated mid-value, along with their dangling key.
func trimTrailingJunk(s string) string {
	s = trimTrailingPunctuation(s)

	// Strip incomplete boolean/null literals. Only partial prefixes are stripped;
	// complete literals (true, false, null) are valid JSON and left alone.
	incompleteLiterals := []string{"tru", "tr", "fals", "fal", "fa", "nul", "nu"}
	strippedLiteral := false
	for _, lit := range incompleteLiterals {
		if strings.HasSuffix(s, lit) {
			s = s[:len(s)-len(lit)]
			strippedLiteral = true
			break
		}
	}

	if !strippedLiteral {
		return s
	}

	// After stripping the literal, remove any trailing colon (which preceded the value).
	s = trimTrailingPunctuation(s)

	// If we stripped `value` from `"key":value`, the trailing `"key"` is now a dangling
	// string. Strip it (along with its preceding comma) to keep JSON valid.
	// Only do this when we actually stripped a literal above, to avoid removing
	// valid array elements.
	if len(s) >= 2 && s[len(s)-1] == '"' {
		openIdx := strings.LastIndex(s[:len(s)-1], `"`)
		if openIdx >= 0 {
			before := openIdx - 1
			if before >= 0 && (s[before] == ',' || s[before] == '{') {
				s = s[:openIdx]
				if len(s) > 0 && s[len(s)-1] == ',' {
					s = s[:len(s)-1]
				}
			}
		}
	}

	return s
}

// trimTrailingPunctuation strips trailing commas and colons.
func trimTrailingPunctuation(s string) string {
	for len(s) > 0 {
		last := s[len(s)-1]
		if last == ',' || last == ':' {
			s = s[:len(s)-1]
			continue
		}
		break
	}
	return s
}
