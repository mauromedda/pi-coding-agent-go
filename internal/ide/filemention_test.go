// ABOUTME: Tests for @file#line-line parsing and content extraction
// ABOUTME: Uses temp files for realistic path resolution testing

package ide

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestParseMentions_NoMentions(t *testing.T) {
	t.Parallel()

	cleaned, mentions, err := ParseMentions("just a normal prompt", "/tmp")
	if err != nil {
		t.Fatal(err)
	}
	if len(mentions) != 0 {
		t.Errorf("expected no mentions, got %d", len(mentions))
	}
	if cleaned != "just a normal prompt" {
		t.Errorf("cleaned = %q, expected unchanged", cleaned)
	}
}

func TestParseMentions_WithFile(t *testing.T) {
	dir := t.TempDir()
	content := "line1\nline2\nline3\nline4\nline5\n"
	path := filepath.Join(dir, "test.go")
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}

	input := "explain @test.go#2-4"
	cleaned, mentions, err := ParseMentions(input, dir)
	if err != nil {
		t.Fatal(err)
	}

	if len(mentions) != 1 {
		t.Fatalf("expected 1 mention, got %d", len(mentions))
	}

	m := mentions[0]
	if m.StartLine != 2 || m.EndLine != 4 {
		t.Errorf("lines = %d-%d, want 2-4", m.StartLine, m.EndLine)
	}
	if !strings.Contains(cleaned, "line2") {
		t.Error("expected cleaned to contain file content")
	}
}

func TestParseRef_PathOnly(t *testing.T) {
	t.Parallel()

	m, err := parseRef("src/main.go", "/project")
	if err != nil {
		t.Fatal(err)
	}
	if m.StartLine != 0 || m.EndLine != 0 {
		t.Errorf("expected no line range, got %d-%d", m.StartLine, m.EndLine)
	}
}

func TestParseRef_WithLines(t *testing.T) {
	t.Parallel()

	m, err := parseRef("src/main.go#10-20", "/project")
	if err != nil {
		t.Fatal(err)
	}
	if m.StartLine != 10 || m.EndLine != 20 {
		t.Errorf("lines = %d-%d, want 10-20", m.StartLine, m.EndLine)
	}
}

func TestUnifiedDiff(t *testing.T) {
	t.Parallel()

	old := "line1\nline2\nline3"
	new := "line1\nmodified\nline3"

	diff := UnifiedDiff("test.go", old, new)

	if !strings.Contains(diff, "-line2") {
		t.Error("expected removed line in diff")
	}
	if !strings.Contains(diff, "+modified") {
		t.Error("expected added line in diff")
	}
}
