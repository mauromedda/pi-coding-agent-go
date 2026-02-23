// ABOUTME: Content truncation with dual line+byte limits and UTF-8 safe boundaries
// ABOUTME: TruncateHead keeps beginning; TruncateTail keeps end (for bash output)

package tools

import (
	"strings"
	"unicode/utf8"
)

const (
	DefaultMaxLines = 2000
	DefaultMaxBytes = 50 * 1024 // 50KB
)

// TruncateResult holds the outcome of a truncation operation.
type TruncateResult struct {
	Content    string
	Truncated  bool
	TotalLines int
	TotalBytes int
	Reason     string // "line_limit", "byte_limit", or ""
}

// TruncateHead keeps the first maxLines lines and first maxBytes bytes.
// If both limits are exceeded, byte_limit wins (tighter constraint).
// The result is always valid UTF-8.
func TruncateHead(content string, maxLines int, maxBytes int) TruncateResult {
	totalBytes := len(content)
	lines := strings.Split(content, "\n")
	totalLines := len(lines)

	result := TruncateResult{
		Content:    content,
		TotalLines: totalLines,
		TotalBytes: totalBytes,
	}

	if content == "" {
		return result
	}

	truncated := false
	reason := ""

	// Apply line limit first.
	if totalLines > maxLines {
		lines = lines[:maxLines]
		result.Content = strings.Join(lines, "\n")
		truncated = true
		reason = "line_limit"
	}

	// Apply byte limit (overrides line limit if tighter).
	if len(result.Content) > maxBytes {
		result.Content = truncateToUTF8Boundary(result.Content, maxBytes)
		truncated = true
		reason = "byte_limit"
	}

	result.Truncated = truncated
	result.Reason = reason
	return result
}

// TruncateTail keeps the last maxLines lines and last maxBytes bytes.
// If both limits are exceeded, byte_limit wins (tighter constraint).
// The result is always valid UTF-8.
func TruncateTail(content string, maxLines int, maxBytes int) TruncateResult {
	totalBytes := len(content)
	lines := strings.Split(content, "\n")
	totalLines := len(lines)

	result := TruncateResult{
		Content:    content,
		TotalLines: totalLines,
		TotalBytes: totalBytes,
	}

	if content == "" {
		return result
	}

	truncated := false
	reason := ""

	// Apply line limit: keep last maxLines lines.
	if totalLines > maxLines {
		lines = lines[totalLines-maxLines:]
		result.Content = strings.Join(lines, "\n")
		truncated = true
		reason = "line_limit"
	}

	// Apply byte limit: keep last maxBytes bytes.
	if len(result.Content) > maxBytes {
		result.Content = truncateTailToUTF8Boundary(result.Content, maxBytes)
		truncated = true
		reason = "byte_limit"
	}

	result.Truncated = truncated
	result.Reason = reason
	return result
}

// truncateToUTF8Boundary truncates content to at most maxBytes,
// walking backward from the cut point to avoid splitting a multi-byte rune.
func truncateToUTF8Boundary(s string, maxBytes int) string {
	if maxBytes >= len(s) {
		return s
	}
	// Walk backward from maxBytes to find a valid rune start.
	for maxBytes > 0 && !utf8.RuneStart(s[maxBytes]) {
		maxBytes--
	}
	return s[:maxBytes]
}

// truncateTailToUTF8Boundary keeps the last maxBytes bytes of content,
// walking forward from the cut point to find a valid rune start.
func truncateTailToUTF8Boundary(s string, maxBytes int) string {
	if maxBytes >= len(s) {
		return s
	}
	start := len(s) - maxBytes
	// Walk forward to find a valid rune start.
	for start < len(s) && !utf8.RuneStart(s[start]) {
		start++
	}
	return s[start:]
}
