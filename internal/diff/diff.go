// ABOUTME: Shared diff generation utilities for text comparison
// ABOUTME: Provides simple and unified diff formats used by tools and IDE

package diff

import (
	"fmt"
	"strings"
)

// Simple produces a minimal line-by-line diff without hunk headers.
// Used for tool output where compact format is preferred.
func Simple(path, before, after string) string {
	oldLines := strings.Split(before, "\n")
	newLines := strings.Split(after, "\n")

	var b strings.Builder
	fmt.Fprintf(&b, "--- %s\n+++ %s\n", path, path)

	maxLen := max(len(newLines), len(oldLines))

	for i := range maxLen {
		oldLine := lineAt(oldLines, i)
		newLine := lineAt(newLines, i)
		if oldLine != newLine {
			if i < len(oldLines) {
				fmt.Fprintf(&b, "-%s\n", oldLine)
			}
			if i < len(newLines) {
				fmt.Fprintf(&b, "+%s\n", newLine)
			}
		}
	}

	return b.String()
}

// Unified generates a unified diff with hunk headers.
// Used for IDE display where standard diff format is expected.
func Unified(path, oldContent, newContent string) string {
	oldLines := strings.Split(oldContent, "\n")
	newLines := strings.Split(newContent, "\n")

	var b strings.Builder
	b.WriteString(fmt.Sprintf("--- a/%s\n", path))
	b.WriteString(fmt.Sprintf("+++ b/%s\n", path))

	maxLen := max(len(newLines), len(oldLines))

	hunkStart := -1
	var hunk strings.Builder

	for i := range maxLen {
		var oldLine, newLine string
		if i < len(oldLines) {
			oldLine = oldLines[i]
		}
		if i < len(newLines) {
			newLine = newLines[i]
		}

		if oldLine != newLine || i >= len(oldLines) || i >= len(newLines) {
			if hunkStart < 0 {
				hunkStart = i
			}
			if i < len(oldLines) {
				hunk.WriteString("-" + oldLine + "\n")
			}
			if i < len(newLines) {
				hunk.WriteString("+" + newLine + "\n")
			}
		} else if hunkStart >= 0 {
			b.WriteString(fmt.Sprintf("@@ -%d +%d @@\n", hunkStart+1, hunkStart+1))
			b.WriteString(hunk.String())
			hunkStart = -1
			hunk.Reset()
		}
	}

	if hunkStart >= 0 {
		b.WriteString(fmt.Sprintf("@@ -%d +%d @@\n", hunkStart+1, hunkStart+1))
		b.WriteString(hunk.String())
	}

	return b.String()
}

// lineAt safely returns the line at index i, or empty string if out of range.
func lineAt(lines []string, i int) string {
	if i < len(lines) {
		return lines[i]
	}
	return ""
}
