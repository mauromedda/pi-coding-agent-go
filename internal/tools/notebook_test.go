// ABOUTME: Tests for the notebook_edit tool: replace, insert, delete cells in .ipynb files
// ABOUTME: Verifies cell operations, bounds checking, error handling, and metadata preservation

package tools

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

// sampleNotebook returns a minimal valid .ipynb JSON string with two cells.
func sampleNotebook() string {
	return `{
 "cells": [
  {
   "cell_type": "code",
   "source": [
    "print('hello')\n"
   ],
   "metadata": {},
   "outputs": []
  },
  {
   "cell_type": "markdown",
   "source": [
    "# Title\n"
   ],
   "metadata": {}
  }
 ],
 "metadata": {
  "kernelspec": {
   "display_name": "Python 3",
   "language": "python",
   "name": "python3"
  }
 },
 "nbformat": 4,
 "nbformat_minor": 5
}`
}

// writeNotebook writes the sample notebook to a temp file and returns the path.
func writeNotebook(t *testing.T, dir string) string {
	t.Helper()
	path := filepath.Join(dir, "test.ipynb")
	if err := os.WriteFile(path, []byte(sampleNotebook()), 0o644); err != nil {
		t.Fatalf("writing sample notebook: %v", err)
	}
	return path
}

// readNotebookCells reads the notebook file and returns the cells slice.
func readNotebookCells(t *testing.T, path string) []map[string]any {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("reading notebook: %v", err)
	}
	var nb map[string]any
	if err := json.Unmarshal(data, &nb); err != nil {
		t.Fatalf("parsing notebook JSON: %v", err)
	}
	cells, ok := nb["cells"].([]any)
	if !ok {
		t.Fatal("cells is not an array")
	}
	result := make([]map[string]any, len(cells))
	for i, c := range cells {
		result[i], ok = c.(map[string]any)
		if !ok {
			t.Fatalf("cell %d is not a map", i)
		}
	}
	return result
}

func TestNotebookEdit_Replace(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	path := writeNotebook(t, dir)

	tool := NewNotebookEditTool()
	result, err := tool.Execute(context.Background(), "id1", map[string]any{
		"path":        path,
		"cell_number": float64(0),
		"operation":   "replace",
		"cell_type":   "code",
		"source":      "x = 42",
	}, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.IsError {
		t.Fatalf("unexpected tool error: %s", result.Content)
	}

	cells := readNotebookCells(t, path)
	if len(cells) != 2 {
		t.Fatalf("expected 2 cells, got %d", len(cells))
	}

	// Verify source was replaced
	src, ok := cells[0]["source"].([]any)
	if !ok {
		t.Fatal("source is not an array")
	}
	if len(src) != 1 || src[0] != "x = 42" {
		t.Errorf("expected source [\"x = 42\"], got %v", src)
	}
}

func TestNotebookEdit_Insert(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	path := writeNotebook(t, dir)

	tool := NewNotebookEditTool()
	result, err := tool.Execute(context.Background(), "id1", map[string]any{
		"path":        path,
		"cell_number": float64(0),
		"operation":   "insert",
		"cell_type":   "code",
		"source":      "y = 99",
	}, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.IsError {
		t.Fatalf("unexpected tool error: %s", result.Content)
	}

	cells := readNotebookCells(t, path)
	if len(cells) != 3 {
		t.Fatalf("expected 3 cells after insert, got %d", len(cells))
	}

	// New cell should be at index 1 (after cell 0)
	src, ok := cells[1]["source"].([]any)
	if !ok {
		t.Fatal("inserted cell source is not an array")
	}
	if len(src) != 1 || src[0] != "y = 99" {
		t.Errorf("expected source [\"y = 99\"], got %v", src)
	}
}

func TestNotebookEdit_Delete(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	path := writeNotebook(t, dir)

	tool := NewNotebookEditTool()
	result, err := tool.Execute(context.Background(), "id1", map[string]any{
		"path":        path,
		"cell_number": float64(1),
		"operation":   "delete",
	}, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.IsError {
		t.Fatalf("unexpected tool error: %s", result.Content)
	}

	cells := readNotebookCells(t, path)
	if len(cells) != 1 {
		t.Fatalf("expected 1 cell after delete, got %d", len(cells))
	}

	// Remaining cell should be the original code cell
	if cells[0]["cell_type"] != "code" {
		t.Errorf("expected remaining cell to be code, got %v", cells[0]["cell_type"])
	}
}

func TestNotebookEdit_OutOfBounds(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	path := writeNotebook(t, dir)

	tool := NewNotebookEditTool()

	tests := []struct {
		name      string
		cellNum   float64
		operation string
	}{
		{"replace negative", float64(-1), "replace"},
		{"replace too high", float64(99), "replace"},
		{"delete too high", float64(5), "delete"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			result, err := tool.Execute(context.Background(), "id1", map[string]any{
				"path":        path,
				"cell_number": tt.cellNum,
				"operation":   tt.operation,
				"source":      "x = 1",
			}, nil)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if !result.IsError {
				t.Error("expected IsError for out-of-bounds cell_number")
			}
		})
	}
}

func TestNotebookEdit_InvalidFile(t *testing.T) {
	t.Parallel()

	tool := NewNotebookEditTool()
	result, err := tool.Execute(context.Background(), "id1", map[string]any{
		"path":        "/nonexistent/path/notebook.ipynb",
		"cell_number": float64(0),
		"operation":   "replace",
		"source":      "x = 1",
	}, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !result.IsError {
		t.Error("expected IsError for non-existent file")
	}
}

func TestNotebookEdit_PreservesMetadata(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	path := writeNotebook(t, dir)

	tool := NewNotebookEditTool()
	_, err := tool.Execute(context.Background(), "id1", map[string]any{
		"path":        path,
		"cell_number": float64(0),
		"operation":   "replace",
		"cell_type":   "code",
		"source":      "replaced = True",
	}, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("reading notebook: %v", err)
	}
	var nb map[string]any
	if err := json.Unmarshal(data, &nb); err != nil {
		t.Fatalf("parsing notebook: %v", err)
	}

	// Verify notebook-level metadata preserved
	meta, ok := nb["metadata"].(map[string]any)
	if !ok {
		t.Fatal("metadata is not a map")
	}
	ks, ok := meta["kernelspec"].(map[string]any)
	if !ok {
		t.Fatal("kernelspec is not a map")
	}
	if ks["display_name"] != "Python 3" {
		t.Errorf("expected kernelspec display_name 'Python 3', got %v", ks["display_name"])
	}

	// Verify nbformat preserved
	if nb["nbformat"] != float64(4) {
		t.Errorf("expected nbformat 4, got %v", nb["nbformat"])
	}
	if nb["nbformat_minor"] != float64(5) {
		t.Errorf("expected nbformat_minor 5, got %v", nb["nbformat_minor"])
	}
}
