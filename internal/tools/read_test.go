// ABOUTME: Tests for the read tool: normal reads, offset/limit, and binary detection
// ABOUTME: Uses t.TempDir for isolated filesystem operations

package tools

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/mauromedda/pi-coding-agent-go/internal/agent"
)

func TestReadTool_NormalFile(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	path := filepath.Join(dir, "test.txt")
	content := "line1\nline2\nline3\n"
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}

	tool := NewReadTool()
	result, err := tool.Execute(context.Background(), "id1", map[string]any{"path": path}, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.IsError {
		t.Fatalf("unexpected tool error: %s", result.Content)
	}
	if result.Content != content {
		t.Errorf("got %q, want %q", result.Content, content)
	}
}

func TestReadTool_OffsetAndLimit(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	path := filepath.Join(dir, "lines.txt")
	content := "line0\nline1\nline2\nline3\nline4\n"
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}

	tool := NewReadTool()
	result, err := tool.Execute(context.Background(), "id1", map[string]any{
		"path":   path,
		"offset": float64(1), // JSON numbers arrive as float64
		"limit":  float64(2),
	}, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.IsError {
		t.Fatalf("unexpected tool error: %s", result.Content)
	}

	expected := "line1\nline2\n"
	if result.Content != expected {
		t.Errorf("got %q, want %q", result.Content, expected)
	}
}

func TestReadTool_BinaryDetection(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	path := filepath.Join(dir, "binary.bin")
	data := []byte("hello\x00world")
	if err := os.WriteFile(path, data, 0o644); err != nil {
		t.Fatal(err)
	}

	tool := NewReadTool()
	result, err := tool.Execute(context.Background(), "id1", map[string]any{"path": path}, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !result.IsError {
		t.Error("expected IsError for binary file")
	}
	if !strings.Contains(result.Content, "binary") {
		t.Errorf("expected 'binary' in error message, got %q", result.Content)
	}
}

func TestReadTool_FileNotFound(t *testing.T) {
	t.Parallel()

	tool := NewReadTool()
	result, err := tool.Execute(context.Background(), "id1", map[string]any{
		"path": "/nonexistent/file.txt",
	}, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !result.IsError {
		t.Error("expected IsError for missing file")
	}
}

func TestReadTool_MissingPath(t *testing.T) {
	t.Parallel()

	tool := NewReadTool()
	result, err := tool.Execute(context.Background(), "id1", map[string]any{}, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !result.IsError {
		t.Error("expected IsError for missing path param")
	}
}

func TestReadTool_LargeFileTruncation(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	path := filepath.Join(dir, "large.txt")
	// Create content larger than maxReadOutput (100KB)
	content := strings.Repeat("x", maxReadOutput+1000)
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}

	tool := NewReadTool()
	result, err := tool.Execute(context.Background(), "id1", map[string]any{"path": path}, func(_ agent.ToolUpdate) {})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.IsError {
		t.Fatalf("unexpected tool error: %s", result.Content)
	}
	if !strings.Contains(result.Content, "truncated") {
		t.Error("expected truncation notice in output")
	}
}
