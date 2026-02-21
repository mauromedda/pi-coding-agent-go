// ABOUTME: Tests for the generic undo/redo stack
// ABOUTME: Covers push, undo, redo, overflow, and edge cases

package undo

import "testing"

func TestStack_UndoRedo(t *testing.T) {
	t.Parallel()

	s := New[string](10)
	s.Push("state1")
	s.Push("state2")

	got, ok := s.Undo()
	if !ok || got != "state2" {
		t.Errorf("Undo() = (%q, %v), want (%q, true)", got, ok, "state2")
	}

	got, ok = s.Redo()
	if !ok || got != "state2" {
		t.Errorf("Redo() = (%q, %v), want (%q, true)", got, ok, "state2")
	}
}

func TestStack_UndoEmpty(t *testing.T) {
	t.Parallel()

	s := New[int](5)
	_, ok := s.Undo()
	if ok {
		t.Error("Undo() on empty stack should return false")
	}
}

func TestStack_RedoClearedOnPush(t *testing.T) {
	t.Parallel()

	s := New[string](10)
	s.Push("a")
	s.Push("b")
	s.Undo()
	s.Push("c")

	if s.CanRedo() {
		t.Error("redo should be cleared after push")
	}
}

func TestStack_Overflow(t *testing.T) {
	t.Parallel()

	s := New[int](3)
	s.Push(1)
	s.Push(2)
	s.Push(3)
	s.Push(4) // Should evict 1

	got, ok := s.Undo()
	if !ok || got != 4 {
		t.Errorf("expected 4, got %d", got)
	}
	got, ok = s.Undo()
	if !ok || got != 3 {
		t.Errorf("expected 3, got %d", got)
	}
	got, ok = s.Undo()
	if !ok || got != 2 {
		t.Errorf("expected 2, got %d", got)
	}
	_, ok = s.Undo()
	if ok {
		t.Error("expected no more undo states")
	}
}
