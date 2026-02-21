// ABOUTME: Tests for AutoMemory: save, load, delete, list, key validation
// ABOUTME: Uses t.TempDir for isolated filesystem; verifies file-based memory persistence

package memory

import (
	"os"
	"path/filepath"
	"testing"
)

func TestAutoMemory_SaveLoad(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	am := NewAutoMemory(dir)

	if err := am.Save("project-context", "This project uses Go 1.26"); err != nil {
		t.Fatalf("Save: %v", err)
	}

	entries, err := am.Load()
	if err != nil {
		t.Fatalf("Load: %v", err)
	}

	if len(entries) != 1 {
		t.Fatalf("expected 1 entry, got %d", len(entries))
	}
	if entries[0].Content != "This project uses Go 1.26" {
		t.Errorf("content mismatch: got %q", entries[0].Content)
	}
	if entries[0].Level != AutoMemory {
		t.Errorf("expected level AutoMemory, got %d", entries[0].Level)
	}
}

func TestAutoMemory_SaveOverwrite(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	am := NewAutoMemory(dir)

	if err := am.Save("key1", "original"); err != nil {
		t.Fatalf("Save original: %v", err)
	}
	if err := am.Save("key1", "updated"); err != nil {
		t.Fatalf("Save updated: %v", err)
	}

	entries, err := am.Load()
	if err != nil {
		t.Fatalf("Load: %v", err)
	}

	if len(entries) != 1 {
		t.Fatalf("expected 1 entry after overwrite, got %d", len(entries))
	}
	if entries[0].Content != "updated" {
		t.Errorf("expected overwritten content %q, got %q", "updated", entries[0].Content)
	}
}

func TestAutoMemory_Delete(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	am := NewAutoMemory(dir)

	if err := am.Save("to-delete", "temporary"); err != nil {
		t.Fatalf("Save: %v", err)
	}

	if err := am.Delete("to-delete"); err != nil {
		t.Fatalf("Delete: %v", err)
	}

	entries, err := am.Load()
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if len(entries) != 0 {
		t.Errorf("expected 0 entries after delete, got %d", len(entries))
	}

	// Verify file is gone
	path := filepath.Join(dir, "to-delete.md")
	if _, err := os.Stat(path); !os.IsNotExist(err) {
		t.Errorf("expected file to be deleted, but it exists")
	}
}

func TestAutoMemory_List(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	am := NewAutoMemory(dir)

	for _, key := range []string{"charlie", "alpha", "bravo"} {
		if err := am.Save(key, "content-"+key); err != nil {
			t.Fatalf("Save %s: %v", key, err)
		}
	}

	keys, err := am.List()
	if err != nil {
		t.Fatalf("List: %v", err)
	}

	if len(keys) != 3 {
		t.Fatalf("expected 3 keys, got %d", len(keys))
	}

	// Keys must be sorted
	expected := []string{"alpha", "bravo", "charlie"}
	for i, want := range expected {
		if keys[i] != want {
			t.Errorf("keys[%d] = %q, want %q", i, keys[i], want)
		}
	}
}

func TestAutoMemory_InvalidKey(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	am := NewAutoMemory(dir)

	tests := []struct {
		name string
		key  string
	}{
		{"slash", "path/traversal"},
		{"dotdot", ".."},
		{"backslash", "back\\slash"},
		{"dotdot-prefix", "../escape"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			err := am.Save(tt.key, "bad")
			if err == nil {
				t.Error("expected error for invalid key")
			}
		})
	}
}

func TestAutoMemory_EmptyDir(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	am := NewAutoMemory(dir)

	entries, err := am.Load()
	if err != nil {
		t.Fatalf("Load on empty dir: %v", err)
	}
	if len(entries) != 0 {
		t.Errorf("expected 0 entries on empty dir, got %d", len(entries))
	}

	keys, err := am.List()
	if err != nil {
		t.Fatalf("List on empty dir: %v", err)
	}
	if len(keys) != 0 {
		t.Errorf("expected 0 keys on empty dir, got %d", len(keys))
	}
}
