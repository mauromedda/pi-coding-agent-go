// ABOUTME: Diff rendering utilities for edit tool output
// ABOUTME: Colors unified diff lines (red/green) and computes simple line diffs

package btea

import (
	"strings"
)

// RenderDiff takes a unified diff string and returns it with ANSI color coding
// derived from the active theme palette via ThemeStyles.
// Added lines use Success, removed lines use Error, headers use Info, hunks use Secondary.
func RenderDiff(diff string, s ThemeStyles) string {
	if diff == "" {
		return ""
	}

	lines := strings.Split(diff, "\n")
	var b strings.Builder
	b.Grow(len(diff) + len(lines)*10) // pre-alloc for ANSI codes

	for _, line := range lines {
		switch {
		case strings.HasPrefix(line, "+++") || strings.HasPrefix(line, "---"):
			b.WriteString(s.DiffHeader.Render(line))
		case strings.HasPrefix(line, "@@"):
			b.WriteString(s.DiffHunk.Render(line))
		case strings.HasPrefix(line, "+"):
			b.WriteString(s.DiffAdded.Render(line))
		case strings.HasPrefix(line, "-"):
			b.WriteString(s.DiffRemoved.Render(line))
		default:
			b.WriteString(line)
		}
		b.WriteByte('\n')
	}

	return strings.TrimRight(b.String(), "\n")
}

// IsEditTool returns true if the tool name is a file-editing tool.
func IsEditTool(name string) bool {
	lower := strings.ToLower(name)
	return lower == "edit" || lower == "write" || lower == "notebookedit"
}

// ComputeSimpleDiff produces a minimal unified-style diff between before and after text.
// Returns an empty string if the texts are identical.
func ComputeSimpleDiff(before, after string) string {
	if before == after {
		return ""
	}

	beforeLines := strings.Split(before, "\n")
	afterLines := strings.Split(after, "\n")

	var b strings.Builder

	// Simple line-by-line comparison (not a real LCS diff, but sufficient for display)
	maxLen := max(len(afterLines), len(beforeLines))

	for i := range maxLen {
		var bLine, aLine string
		hasBefore := i < len(beforeLines)
		hasAfter := i < len(afterLines)

		if hasBefore {
			bLine = beforeLines[i]
		}
		if hasAfter {
			aLine = afterLines[i]
		}

		if hasBefore && hasAfter && bLine == aLine {
			b.WriteString(" " + bLine + "\n")
		} else {
			if hasBefore {
				b.WriteString("-" + bLine + "\n")
			}
			if hasAfter {
				b.WriteString("+" + aLine + "\n")
			}
		}
	}

	return strings.TrimRight(b.String(), "\n")
}
