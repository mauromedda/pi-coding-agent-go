// ABOUTME: Tests for the write tool: file creation, overwrite, and directory creation
// ABOUTME: Uses t.TempDir for isolated filesystem operations

package tools

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestWriteTool_CreateNewFile(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	path := filepath.Join(dir, "new.txt")

	tool := NewWriteTool()
	result, err := tool.Execute(context.Background(), "id1", map[string]any{
		"path":    path,
		"content": "hello world",
	}, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.IsError {
		t.Fatalf("unexpected tool error: %s", result.Content)
	}

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("reading written file: %v", err)
	}
	if string(data) != "hello world" {
		t.Errorf("got %q, want %q", string(data), "hello world")
	}
}

func TestWriteTool_OverwriteExisting(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	path := filepath.Join(dir, "existing.txt")
	if err := os.WriteFile(path, []byte("old"), 0o644); err != nil {
		t.Fatal(err)
	}

	tool := NewWriteTool()
	result, err := tool.Execute(context.Background(), "id1", map[string]any{
		"path":    path,
		"content": "new",
	}, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.IsError {
		t.Fatalf("unexpected tool error: %s", result.Content)
	}

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("reading file: %v", err)
	}
	if string(data) != "new" {
		t.Errorf("got %q, want %q", string(data), "new")
	}
}

func TestWriteTool_CreatesParentDirs(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	path := filepath.Join(dir, "a", "b", "c", "deep.txt")

	tool := NewWriteTool()
	result, err := tool.Execute(context.Background(), "id1", map[string]any{
		"path":    path,
		"content": "deep content",
	}, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.IsError {
		t.Fatalf("unexpected tool error: %s", result.Content)
	}

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("reading file: %v", err)
	}
	if string(data) != "deep content" {
		t.Errorf("got %q, want %q", string(data), "deep content")
	}
}

func TestWriteTool_MissingParams(t *testing.T) {
	t.Parallel()

	tool := NewWriteTool()

	tests := []struct {
		name   string
		params map[string]any
	}{
		{"missing path", map[string]any{"content": "x"}},
		{"missing content", map[string]any{"path": "/tmp/test.txt"}},
		{"empty params", map[string]any{}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			result, err := tool.Execute(context.Background(), "id1", tt.params, nil)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if !result.IsError {
				t.Error("expected IsError for missing params")
			}
		})
	}
}

func TestWriteTool_ResultContainsByteCount(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	path := filepath.Join(dir, "count.txt")

	tool := NewWriteTool()
	result, err := tool.Execute(context.Background(), "id1", map[string]any{
		"path":    path,
		"content": "12345",
	}, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(result.Content, "5 bytes") {
		t.Errorf("expected byte count in result, got %q", result.Content)
	}
}
