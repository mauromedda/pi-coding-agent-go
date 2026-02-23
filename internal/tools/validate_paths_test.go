// ABOUTME: Tests for validate_paths tool: checks existence and type of a list of paths
// ABOUTME: Covers found/not-found, mixed results, empty input, and summary output

package tools

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestValidatePaths_AllExist(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	f1 := filepath.Join(dir, "a.go")
	os.WriteFile(f1, []byte("package a"), 0o644)
	sub := filepath.Join(dir, "sub")
	os.Mkdir(sub, 0o755)

	tool := NewValidatePathsTool()
	params := map[string]any{"paths": []any{f1, sub}}

	result, err := tool.Execute(context.Background(), "", params, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.IsError {
		t.Fatalf("unexpected tool error: %s", result.Content)
	}
	if !strings.Contains(result.Content, "2/2 paths exist") {
		t.Errorf("expected summary '2/2 paths exist', got:\n%s", result.Content)
	}
	if !strings.Contains(result.Content, "file") {
		t.Error("expected 'file' type in output")
	}
	if !strings.Contains(result.Content, "dir") {
		t.Error("expected 'dir' type in output")
	}
}

func TestValidatePaths_MixedExistence(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	existing := filepath.Join(dir, "exists.txt")
	os.WriteFile(existing, []byte("hi"), 0o644)
	missing := filepath.Join(dir, "nope.txt")

	tool := NewValidatePathsTool()
	params := map[string]any{"paths": []any{existing, missing}}

	result, err := tool.Execute(context.Background(), "", params, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(result.Content, "1/2 paths exist") {
		t.Errorf("expected '1/2 paths exist', got:\n%s", result.Content)
	}
	if !strings.Contains(result.Content, "not found") {
		t.Error("expected 'not found' in output")
	}
}

func TestValidatePaths_MissingParam(t *testing.T) {
	t.Parallel()

	tool := NewValidatePathsTool()
	result, err := tool.Execute(context.Background(), "", map[string]any{}, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !result.IsError {
		t.Error("expected IsError for missing param")
	}
}

func TestValidatePaths_IsReadOnly(t *testing.T) {
	t.Parallel()

	tool := NewValidatePathsTool()
	if !tool.ReadOnly {
		t.Error("validate_paths must be read-only")
	}
}
