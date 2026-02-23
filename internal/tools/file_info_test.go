// ABOUTME: Tests for file_info tool: metadata extraction without reading content
// ABOUTME: Covers regular files, directories, binary detection, language mapping

package tools

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestFileInfo_RegularFile(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	p := filepath.Join(dir, "main.go")
	os.WriteFile(p, []byte("package main\nfunc main() {}\n"), 0o644)

	tool := NewFileInfoTool()
	result, err := tool.Execute(context.Background(), "", map[string]any{"path": p}, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.IsError {
		t.Fatalf("unexpected tool error: %s", result.Content)
	}

	for _, want := range []string{"lines: 2", "language: go", "binary: false"} {
		if !strings.Contains(result.Content, want) {
			t.Errorf("expected %q in output:\n%s", want, result.Content)
		}
	}
}

func TestFileInfo_Directory(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	tool := NewFileInfoTool()
	result, err := tool.Execute(context.Background(), "", map[string]any{"path": dir}, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(result.Content, "type: directory") {
		t.Errorf("expected 'type: directory' in output:\n%s", result.Content)
	}
}

func TestFileInfo_BinaryFile(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	p := filepath.Join(dir, "data.bin")
	// Write bytes with many nulls to trigger binary detection.
	data := make([]byte, 512)
	for i := 0; i < 256; i++ {
		data[i*2] = 0
	}
	os.WriteFile(p, data, 0o644)

	tool := NewFileInfoTool()
	result, err := tool.Execute(context.Background(), "", map[string]any{"path": p}, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(result.Content, "binary: true") {
		t.Errorf("expected 'binary: true' in output:\n%s", result.Content)
	}
}

func TestFileInfo_NotFound(t *testing.T) {
	t.Parallel()

	tool := NewFileInfoTool()
	result, err := tool.Execute(context.Background(), "", map[string]any{"path": "/no/such/file"}, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !result.IsError {
		t.Error("expected IsError for missing file")
	}
}

func TestFileInfo_MissingParam(t *testing.T) {
	t.Parallel()

	tool := NewFileInfoTool()
	result, err := tool.Execute(context.Background(), "", map[string]any{}, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !result.IsError {
		t.Error("expected IsError for missing param")
	}
}

func TestFileInfo_IsReadOnly(t *testing.T) {
	t.Parallel()

	tool := NewFileInfoTool()
	if !tool.ReadOnly {
		t.Error("file_info must be read-only")
	}
}
