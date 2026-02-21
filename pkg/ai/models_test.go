// ABOUTME: Tests for model lookup by ID using pre-built index
// ABOUTME: Covers found, not-found cases and benchmark for O(1) lookup

package ai

import "testing"

func TestFindModel_Found(t *testing.T) {
	t.Parallel()

	m := FindModel("claude-opus-4-20250514")
	if m == nil {
		t.Fatal("expected non-nil model")
	}
	if m.Name != "Claude Opus 4" {
		t.Errorf("Name = %q, want %q", m.Name, "Claude Opus 4")
	}
}

func TestFindModel_NotFound(t *testing.T) {
	t.Parallel()

	m := FindModel("nonexistent-model")
	if m != nil {
		t.Errorf("expected nil for unknown model, got %v", m)
	}
}

func BenchmarkFindModel(b *testing.B) {
	for b.Loop() {
		FindModel("claude-opus-4-20250514")
	}
}
