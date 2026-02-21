// ABOUTME: IDE-aware diff output generation
// ABOUTME: Produces unified diff format for terminal display

package ide

import (
	"fmt"
	"strings"
)

// UnifiedDiff generates a unified diff between old and new content.
func UnifiedDiff(path, oldContent, newContent string) string {
	oldLines := strings.Split(oldContent, "\n")
	newLines := strings.Split(newContent, "\n")

	var b strings.Builder
	b.WriteString(fmt.Sprintf("--- a/%s\n", path))
	b.WriteString(fmt.Sprintf("+++ b/%s\n", path))

	// Simple line-by-line diff (not a full Myers diff, but functional)
	maxLen := len(oldLines)
	if len(newLines) > maxLen {
		maxLen = len(newLines)
	}

	hunkStart := -1
	var hunk strings.Builder

	for i := 0; i < maxLen; i++ {
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
			// Flush hunk
			b.WriteString(fmt.Sprintf("@@ -%d +%d @@\n", hunkStart+1, hunkStart+1))
			b.WriteString(hunk.String())
			hunkStart = -1
			hunk.Reset()
		}
	}

	// Flush final hunk
	if hunkStart >= 0 {
		b.WriteString(fmt.Sprintf("@@ -%d +%d @@\n", hunkStart+1, hunkStart+1))
		b.WriteString(hunk.String())
	}

	return b.String()
}
