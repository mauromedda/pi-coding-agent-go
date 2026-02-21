// ABOUTME: IDE-aware diff output generation
// ABOUTME: Delegates to shared diff package for unified diff format

package ide

import "github.com/mauromedda/pi-coding-agent-go/internal/diff"

// UnifiedDiff generates a unified diff between old and new content.
func UnifiedDiff(path, oldContent, newContent string) string {
	return diff.Unified(path, oldContent, newContent)
}
