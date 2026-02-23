// ABOUTME: Tests for checkpoint metadata persistence
// ABOUTME: Uses temp directories for isolated file-based tests

package ide

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestMetadataStore_SaveAndList(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	store, err := NewMetadataStore(dir)
	if err != nil {
		t.Fatalf("NewMetadataStore: %v", err)
	}

	now := time.Now().UTC().Truncate(time.Millisecond)
	records := []CheckpointRecord{
		{ID: "aaa", Ref: "ref1", Timestamp: now.Add(-2 * time.Second), ToolName: "bash", ToolArgs: "ls", SessionID: "s1"},
		{ID: "bbb", Ref: "ref2", Timestamp: now.Add(-1 * time.Second), ToolName: "write", ToolArgs: "f.go", Name: "before refactor", SessionID: "s1"},
		{ID: "ccc", Ref: "ref3", Timestamp: now, ToolName: "edit", ToolArgs: "main.go", Description: "desc", Mode: "execute", SessionID: "s1"},
	}

	for _, r := range records {
		if err := store.Save(r); err != nil {
			t.Fatalf("Save(%s): %v", r.ID, err)
		}
	}

	got, err := store.List()
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if len(got) != 3 {
		t.Fatalf("List returned %d records; want 3", len(got))
	}

	// Newest first
	if got[0].ID != "ccc" {
		t.Errorf("got[0].ID = %q; want %q", got[0].ID, "ccc")
	}
	if got[1].ID != "bbb" {
		t.Errorf("got[1].ID = %q; want %q", got[1].ID, "bbb")
	}
	if got[2].ID != "aaa" {
		t.Errorf("got[2].ID = %q; want %q", got[2].ID, "aaa")
	}

	// Verify fields round-trip
	if got[0].Mode != "execute" {
		t.Errorf("got[0].Mode = %q; want %q", got[0].Mode, "execute")
	}
	if got[1].Name != "before refactor" {
		t.Errorf("got[1].Name = %q; want %q", got[1].Name, "before refactor")
	}
}

func TestMetadataStore_ListBySession(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	store, err := NewMetadataStore(dir)
	if err != nil {
		t.Fatalf("NewMetadataStore: %v", err)
	}

	now := time.Now().UTC().Truncate(time.Millisecond)
	records := []CheckpointRecord{
		{ID: "r1", Ref: "ref1", Timestamp: now.Add(-2 * time.Second), ToolName: "bash", SessionID: "session-A"},
		{ID: "r2", Ref: "ref2", Timestamp: now.Add(-1 * time.Second), ToolName: "write", SessionID: "session-B"},
		{ID: "r3", Ref: "ref3", Timestamp: now, ToolName: "edit", SessionID: "session-A"},
	}

	for _, r := range records {
		if err := store.Save(r); err != nil {
			t.Fatalf("Save(%s): %v", r.ID, err)
		}
	}

	got, err := store.ListBySession("session-A")
	if err != nil {
		t.Fatalf("ListBySession: %v", err)
	}
	if len(got) != 2 {
		t.Fatalf("ListBySession returned %d; want 2", len(got))
	}

	// Newest first
	if got[0].ID != "r3" {
		t.Errorf("got[0].ID = %q; want %q", got[0].ID, "r3")
	}
	if got[1].ID != "r1" {
		t.Errorf("got[1].ID = %q; want %q", got[1].ID, "r1")
	}
}

func TestMetadataStore_Delete(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	store, err := NewMetadataStore(dir)
	if err != nil {
		t.Fatalf("NewMetadataStore: %v", err)
	}

	r := CheckpointRecord{
		ID:        "del-me",
		Ref:       "ref1",
		Timestamp: time.Now().UTC().Truncate(time.Millisecond),
		ToolName:  "bash",
		SessionID: "s1",
	}
	if err := store.Save(r); err != nil {
		t.Fatalf("Save: %v", err)
	}

	if err := store.Delete("del-me"); err != nil {
		t.Fatalf("Delete: %v", err)
	}

	got, err := store.List()
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if len(got) != 0 {
		t.Errorf("List returned %d after delete; want 0", len(got))
	}
}

func TestMetadataStore_Delete_NotFound(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	store, err := NewMetadataStore(dir)
	if err != nil {
		t.Fatalf("NewMetadataStore: %v", err)
	}

	err = store.Delete("nonexistent")
	if err == nil {
		t.Error("Delete(nonexistent) should return error")
	}
}

func TestMetadataStore_EmptyDir(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	store, err := NewMetadataStore(dir)
	if err != nil {
		t.Fatalf("NewMetadataStore: %v", err)
	}

	got, err := store.List()
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if len(got) != 0 {
		t.Errorf("List on empty dir returned %d; want 0", len(got))
	}
}

func TestMetadataStore_CreatesDirIfMissing(t *testing.T) {
	t.Parallel()

	base := t.TempDir()
	nested := filepath.Join(base, "sub", "checkpoints")

	store, err := NewMetadataStore(nested)
	if err != nil {
		t.Fatalf("NewMetadataStore: %v", err)
	}

	r := CheckpointRecord{
		ID:        "nested-1",
		Ref:       "ref1",
		Timestamp: time.Now().UTC().Truncate(time.Millisecond),
		ToolName:  "bash",
		SessionID: "s1",
	}
	if err := store.Save(r); err != nil {
		t.Fatalf("Save: %v", err)
	}

	got, err := store.List()
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if len(got) != 1 {
		t.Fatalf("List returned %d; want 1", len(got))
	}

	_ = store // silence
}

func TestMetadataStore_InvalidJSON(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()

	// Write a corrupt file
	corrupt := filepath.Join(dir, "20060102T150405Z_bad-id.json")
	if err := os.WriteFile(corrupt, []byte("{invalid json!!!"), 0o644); err != nil {
		t.Fatal(err)
	}

	// Write a valid file
	valid := CheckpointRecord{
		ID:        "good-id",
		Ref:       "ref1",
		Timestamp: time.Date(2026, 1, 15, 10, 30, 0, 0, time.UTC),
		ToolName:  "bash",
		SessionID: "s1",
	}
	data, err := json.Marshal(valid)
	if err != nil {
		t.Fatal(err)
	}
	validPath := filepath.Join(dir, "20260115T103000Z_good-id.json")
	if err := os.WriteFile(validPath, data, 0o644); err != nil {
		t.Fatal(err)
	}

	store, err := NewMetadataStore(dir)
	if err != nil {
		t.Fatalf("NewMetadataStore: %v", err)
	}

	got, err := store.List()
	if err != nil {
		t.Fatalf("List should not error on corrupt files: %v", err)
	}
	if len(got) != 1 {
		t.Fatalf("List returned %d; want 1 (corrupt file skipped)", len(got))
	}
	if got[0].ID != "good-id" {
		t.Errorf("got[0].ID = %q; want %q", got[0].ID, "good-id")
	}
}
