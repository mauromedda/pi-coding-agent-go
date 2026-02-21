// ABOUTME: Tests for Simple and Unified diff generation
// ABOUTME: Verifies output format, line changes, and edge cases

package diff

import (
	"strings"
	"testing"
)

func TestSimple_BasicChange(t *testing.T) {
	t.Parallel()

	result := Simple("test.txt", "alpha\nbeta\ngamma\n", "alpha\nBETA\ngamma\n")

	if !strings.Contains(result, "-beta") {
		t.Errorf("expected removed line, got %q", result)
	}
	if !strings.Contains(result, "+BETA") {
		t.Errorf("expected added line, got %q", result)
	}
	if !strings.Contains(result, "--- test.txt") {
		t.Errorf("expected header, got %q", result)
	}
}

func TestSimple_NoChange(t *testing.T) {
	t.Parallel()

	result := Simple("file.go", "same\n", "same\n")

	if strings.Contains(result, "-") && !strings.Contains(result, "---") {
		t.Errorf("expected no diff lines for identical content, got %q", result)
	}
}

func TestUnified_HunkHeaders(t *testing.T) {
	t.Parallel()

	result := Unified("file.go", "line1\nline2\nline3\n", "line1\nLINE2\nline3\n")

	if !strings.Contains(result, "@@") {
		t.Errorf("expected hunk header, got %q", result)
	}
	if !strings.Contains(result, "--- a/file.go") {
		t.Errorf("expected unified header format, got %q", result)
	}
}

func TestUnified_MultipleHunks(t *testing.T) {
	t.Parallel()

	old := "a\nb\nc\nd\ne\nf\n"
	new := "A\nb\nc\nd\ne\nF\n"

	result := Unified("test.go", old, new)

	// Should have two hunks (line 1 and line 6 changed)
	count := strings.Count(result, "@@")
	if count < 4 { // Two hunks = 4 @@ markers (2 per hunk header)
		t.Errorf("expected at least 2 hunks, got %d @@ markers in %q", count, result)
	}
}
