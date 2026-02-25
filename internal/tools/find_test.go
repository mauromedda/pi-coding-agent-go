// ABOUTME: Tests for find tool: covers basic glob, doublestar, mod-time sort, head_limit, no-matches
// ABOUTME: Uses builtin path only (no rg dependency in CI)

package tools

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// setupFindTestDir creates a temporary directory with test files for find tests.
func setupFindTestDir(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()

	// Create files with staggered mod times
	files := []struct {
		path    string
		content string
		delay   time.Duration
	}{
		{"old.go", "package old", 0},
		{"mid.py", "# mid", 50 * time.Millisecond},
		{"sub/deep.go", "package sub", 100 * time.Millisecond},
		{"sub/inner/deeper.ts", "export {}", 150 * time.Millisecond},
		{"newest.go", "package newest", 200 * time.Millisecond},
	}

	for _, f := range files {
		full := filepath.Join(dir, f.path)
		if err := os.MkdirAll(filepath.Dir(full), 0o755); err != nil {
			t.Fatal(err)
		}
		if f.delay > 0 {
			time.Sleep(f.delay)
		}
		if err := os.WriteFile(full, []byte(f.content), 0o644); err != nil {
			t.Fatal(err)
		}
	}

	return dir
}

func TestFindBuiltin_BasicGlob(t *testing.T) {
	dir := setupFindTestDir(t)
	out, err := findBuiltin("*.go", dir, 0)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out, ".go") {
		t.Errorf("expected .go files, got:\n%s", out)
	}
	if strings.Contains(out, ".py") {
		t.Errorf("*.go should not match .py files, got:\n%s", out)
	}
}

func TestFindBuiltin_DoubleStarGlob(t *testing.T) {
	dir := setupFindTestDir(t)
	out, err := findBuiltin("**/*.go", dir, 0)
	if err != nil {
		t.Fatal(err)
	}
	lines := strings.Split(strings.TrimSpace(out), "\n")
	// Should find all .go files including in subdirectories
	goCount := 0
	for _, line := range lines {
		if strings.HasSuffix(line, ".go") {
			goCount++
		}
	}
	if goCount < 3 {
		t.Errorf("expected at least 3 .go files with **/*.go, got %d:\n%s", goCount, out)
	}
}

func TestFindBuiltin_ModTimeSortNewestFirst(t *testing.T) {
	dir := setupFindTestDir(t)
	out, err := findBuiltin("**/*.go", dir, 0)
	if err != nil {
		t.Fatal(err)
	}
	lines := strings.Split(strings.TrimSpace(out), "\n")
	if len(lines) < 2 {
		t.Fatalf("expected at least 2 results, got %d", len(lines))
	}
	// newest.go should come first (newest mod time)
	if !strings.HasSuffix(lines[0], "newest.go") {
		t.Errorf("expected newest.go first (newest mod time), got: %s", lines[0])
	}
}

func TestFindBuiltin_HeadLimit(t *testing.T) {
	dir := setupFindTestDir(t)
	out, err := findBuiltin("**/*", dir, 2)
	if err != nil {
		t.Fatal(err)
	}
	lines := strings.Split(strings.TrimSpace(out), "\n")
	if len(lines) > 2 {
		t.Errorf("head_limit=2 should return at most 2 results, got %d:\n%s", len(lines), out)
	}
}

func TestFindBuiltin_NoMatches(t *testing.T) {
	dir := setupFindTestDir(t)
	out, err := findBuiltin("*.xyz", dir, 0)
	if err != nil {
		t.Fatal(err)
	}
	if out != "no files found" {
		t.Errorf("expected 'no files found', got: %s", out)
	}
}

func TestFindBuiltin_SubdirGlob(t *testing.T) {
	dir := setupFindTestDir(t)
	out, err := findBuiltin("sub/**/*.go", dir, 0)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out, "deep.go") {
		t.Errorf("expected sub/deep.go in results, got:\n%s", out)
	}
}
