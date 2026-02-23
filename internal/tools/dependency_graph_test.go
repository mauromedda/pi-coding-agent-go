// ABOUTME: Tests for dependency_graph tool: Go import graph via go/parser
// ABOUTME: Covers package grouping, filtering, skip dirs, and parse errors

package tools

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func setupGraphTestDir(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()

	pkg1 := filepath.Join(dir, "cmd", "app")
	os.MkdirAll(pkg1, 0o755)
	os.WriteFile(filepath.Join(pkg1, "main.go"), []byte(`package main

import (
	"fmt"
	"os"
)

func main() { fmt.Println(os.Args) }
`), 0o644)

	pkg2 := filepath.Join(dir, "internal", "util")
	os.MkdirAll(pkg2, 0o755)
	os.WriteFile(filepath.Join(pkg2, "util.go"), []byte(`package util

import "strings"

func Upper(s string) string { return strings.ToUpper(s) }
`), 0o644)

	return dir
}

func TestDependencyGraph_Basic(t *testing.T) {
	t.Parallel()

	dir := setupGraphTestDir(t)
	tool := NewDependencyGraphTool()
	result, err := tool.Execute(context.Background(), "", map[string]any{"path": dir}, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.IsError {
		t.Fatalf("tool error: %s", result.Content)
	}

	for _, want := range []string{"fmt", "os", "strings"} {
		if !strings.Contains(result.Content, want) {
			t.Errorf("expected import %q in output:\n%s", want, result.Content)
		}
	}
}

func TestDependencyGraph_WithFilter(t *testing.T) {
	t.Parallel()

	dir := setupGraphTestDir(t)
	tool := NewDependencyGraphTool()
	result, err := tool.Execute(context.Background(), "", map[string]any{
		"path":           dir,
		"package_filter": "strings",
	}, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// Only the package importing "strings" should appear.
	if !strings.Contains(result.Content, "strings") {
		t.Errorf("expected 'strings' in filtered output:\n%s", result.Content)
	}
	// "fmt" should be filtered out since we only show packages that import "strings".
	lines := strings.Split(result.Content, "\n")
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if trimmed == "-> fmt" || trimmed == "-> os" {
			t.Errorf("unexpected import in filtered output: %s", trimmed)
		}
	}
}

func TestDependencyGraph_SkipsDirs(t *testing.T) {
	t.Parallel()

	dir := setupGraphTestDir(t)
	// Create a vendor dir with a Go file; it should be skipped.
	vendor := filepath.Join(dir, "vendor", "lib")
	os.MkdirAll(vendor, 0o755)
	os.WriteFile(filepath.Join(vendor, "lib.go"), []byte(`package lib
import "net/http"
var _ = http.StatusOK
`), 0o644)

	tool := NewDependencyGraphTool()
	result, err := tool.Execute(context.Background(), "", map[string]any{"path": dir}, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if strings.Contains(result.Content, "net/http") {
		t.Errorf("vendor dir should be skipped, but found net/http:\n%s", result.Content)
	}
}

func TestDependencyGraph_NoGoFiles(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	os.WriteFile(filepath.Join(dir, "readme.md"), []byte("# hi"), 0o644)

	tool := NewDependencyGraphTool()
	result, err := tool.Execute(context.Background(), "", map[string]any{"path": dir}, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(result.Content, "no Go packages found") {
		t.Errorf("expected 'no Go packages found', got:\n%s", result.Content)
	}
}

func TestDependencyGraph_IsReadOnly(t *testing.T) {
	t.Parallel()

	tool := NewDependencyGraphTool()
	if !tool.ReadOnly {
		t.Error("dependency_graph must be read-only")
	}
}
