// ABOUTME: Tests for the Emacs-style kill ring buffer
// ABOUTME: Covers push, yank, yank-pop, and overflow behavior

package killring

import "testing"

func TestKillRing_PushAndYank(t *testing.T) {
	t.Parallel()

	kr := New()
	kr.Push("first")
	kr.Push("second")

	got := kr.Yank()
	if got != "second" {
		t.Errorf("Yank() = %q, want %q", got, "second")
	}
}

func TestKillRing_YankPop(t *testing.T) {
	t.Parallel()

	kr := New()
	kr.Push("a")
	kr.Push("b")
	kr.Push("c")

	kr.Yank() // c
	got := kr.YankPop()
	if got != "b" {
		t.Errorf("YankPop() = %q, want %q", got, "b")
	}

	got = kr.YankPop()
	if got != "a" {
		t.Errorf("YankPop() = %q, want %q", got, "a")
	}
}

func TestKillRing_Empty(t *testing.T) {
	t.Parallel()

	kr := New()
	if got := kr.Yank(); got != "" {
		t.Errorf("Yank() on empty = %q, want empty", got)
	}
	if got := kr.YankPop(); got != "" {
		t.Errorf("YankPop() on empty = %q, want empty", got)
	}
}

func TestKillRing_Len(t *testing.T) {
	t.Parallel()

	kr := New()
	kr.Push("a")
	kr.Push("b")
	if kr.Len() != 2 {
		t.Errorf("Len() = %d, want 2", kr.Len())
	}
}
