// ABOUTME: Tests for the edit tool: single replace, replace_all, and error cases
// ABOUTME: Uses t.TempDir for isolated filesystem operations

package tools

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestEditTool_SimpleReplace(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	path := filepath.Join(dir, "edit.txt")
	if err := os.WriteFile(path, []byte("hello world"), 0o644); err != nil {
		t.Fatal(err)
	}

	tool := NewEditTool()
	result, err := tool.Execute(context.Background(), "id1", map[string]any{
		"path":       path,
		"old_string": "world",
		"new_string": "Go",
	}, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.IsError {
		t.Fatalf("unexpected tool error: %s", result.Content)
	}

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if string(data) != "hello Go" {
		t.Errorf("got %q, want %q", string(data), "hello Go")
	}
}

func TestEditTool_ReplaceAll(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	path := filepath.Join(dir, "edit.txt")
	if err := os.WriteFile(path, []byte("aaa bbb aaa"), 0o644); err != nil {
		t.Fatal(err)
	}

	tool := NewEditTool()
	result, err := tool.Execute(context.Background(), "id1", map[string]any{
		"path":        path,
		"old_string":  "aaa",
		"new_string":  "ccc",
		"replace_all": true,
	}, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.IsError {
		t.Fatalf("unexpected tool error: %s", result.Content)
	}

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if string(data) != "ccc bbb ccc" {
		t.Errorf("got %q, want %q", string(data), "ccc bbb ccc")
	}
}

func TestEditTool_NotUniqueError(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	path := filepath.Join(dir, "edit.txt")
	if err := os.WriteFile(path, []byte("foo bar foo"), 0o644); err != nil {
		t.Fatal(err)
	}

	tool := NewEditTool()
	result, err := tool.Execute(context.Background(), "id1", map[string]any{
		"path":       path,
		"old_string": "foo",
		"new_string": "baz",
	}, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !result.IsError {
		t.Error("expected IsError when old_string is not unique")
	}
	if !strings.Contains(result.Content, "2 times") {
		t.Errorf("expected count in error message, got %q", result.Content)
	}
}

func TestEditTool_NotFoundError(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	path := filepath.Join(dir, "edit.txt")
	if err := os.WriteFile(path, []byte("hello world"), 0o644); err != nil {
		t.Fatal(err)
	}

	tool := NewEditTool()
	result, err := tool.Execute(context.Background(), "id1", map[string]any{
		"path":       path,
		"old_string": "missing",
		"new_string": "replaced",
	}, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !result.IsError {
		t.Error("expected IsError when old_string not found")
	}
	if !strings.Contains(result.Content, "not found") {
		t.Errorf("expected 'not found' in error, got %q", result.Content)
	}
}

func TestEditTool_DiffOutput(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	path := filepath.Join(dir, "diff.txt")
	if err := os.WriteFile(path, []byte("alpha\nbeta\ngamma\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	tool := NewEditTool()
	result, err := tool.Execute(context.Background(), "id1", map[string]any{
		"path":       path,
		"old_string": "beta",
		"new_string": "BETA",
	}, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.IsError {
		t.Fatalf("unexpected tool error: %s", result.Content)
	}

	if !strings.Contains(result.Content, "-beta") || !strings.Contains(result.Content, "+BETA") {
		t.Errorf("expected diff output with -/+ lines, got %q", result.Content)
	}
}
