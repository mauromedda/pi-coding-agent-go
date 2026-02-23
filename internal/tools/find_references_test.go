// ABOUTME: Tests for find_references tool: symbol usage search across files
// ABOUTME: Covers rg path, stdlib fallback, missing params, and include glob

package tools

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func setupRefTestDir(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()

	os.WriteFile(filepath.Join(dir, "main.go"), []byte(`package main

func main() {
	DoStuff()
}
`), 0o644)

	os.WriteFile(filepath.Join(dir, "stuff.go"), []byte(`package main

func DoStuff() {
	println("hello")
}
`), 0o644)

	os.WriteFile(filepath.Join(dir, "other.py"), []byte(`def helper():
    DoStuff()
`), 0o644)

	return dir
}

func TestFindReferences_StdlibFallback(t *testing.T) {
	t.Parallel()

	dir := setupRefTestDir(t)
	// Force stdlib fallback (hasRg=false).
	tool := NewFindReferencesTool(false)
	params := map[string]any{"symbol": "DoStuff", "path": dir}

	result, err := tool.Execute(context.Background(), "", params, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.IsError {
		t.Fatalf("tool error: %s", result.Content)
	}
	// Should find references in main.go, stuff.go, and other.py.
	if !strings.Contains(result.Content, "main.go") {
		t.Errorf("expected main.go in output:\n%s", result.Content)
	}
	if !strings.Contains(result.Content, "stuff.go") {
		t.Errorf("expected stuff.go in output:\n%s", result.Content)
	}
	if !strings.Contains(result.Content, "other.py") {
		t.Errorf("expected other.py in output:\n%s", result.Content)
	}
}

func TestFindReferences_WithInclude(t *testing.T) {
	t.Parallel()

	dir := setupRefTestDir(t)
	tool := NewFindReferencesTool(false)
	params := map[string]any{"symbol": "DoStuff", "path": dir, "include": "*.go"}

	result, err := tool.Execute(context.Background(), "", params, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// With include=*.go, should NOT include other.py.
	if strings.Contains(result.Content, "other.py") {
		t.Errorf("expected other.py to be excluded with include=*.go:\n%s", result.Content)
	}
	if !strings.Contains(result.Content, "main.go") {
		t.Errorf("expected main.go in output:\n%s", result.Content)
	}
}

func TestFindReferences_NoMatches(t *testing.T) {
	t.Parallel()

	dir := setupRefTestDir(t)
	tool := NewFindReferencesTool(false)
	params := map[string]any{"symbol": "NothingMatchesThis", "path": dir}

	result, err := tool.Execute(context.Background(), "", params, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(result.Content, "no references found") {
		t.Errorf("expected 'no references found', got:\n%s", result.Content)
	}
}

func TestFindReferences_MissingSymbol(t *testing.T) {
	t.Parallel()

	tool := NewFindReferencesTool(false)
	result, err := tool.Execute(context.Background(), "", map[string]any{}, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !result.IsError {
		t.Error("expected IsError for missing symbol param")
	}
}

func TestFindReferences_IsReadOnly(t *testing.T) {
	t.Parallel()

	tool := NewFindReferencesTool(false)
	if !tool.ReadOnly {
		t.Error("find_references must be read-only")
	}
}
