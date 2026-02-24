// ABOUTME: Tests for WorktreeDialogModel overlay: key handling and View rendering
// ABOUTME: Validates Merge/Keep/Discard key bindings and proper message types

package btea

import (
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
)

func TestWorktreeDialogModel_MergeKey(t *testing.T) {
	m := NewWorktreeDialogModel("pi-go/session-test", 80)
	_, cmd := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'m'}})
	if cmd == nil {
		t.Fatal("cmd = nil; want WorktreeExitMsg")
	}
	msg := cmd()
	wm, ok := msg.(WorktreeExitMsg)
	if !ok {
		t.Fatalf("cmd() = %T; want WorktreeExitMsg", msg)
	}
	if wm.Action != WorktreeActionMerge {
		t.Errorf("Action = %d; want WorktreeActionMerge", wm.Action)
	}
}

func TestWorktreeDialogModel_KeepKey(t *testing.T) {
	m := NewWorktreeDialogModel("pi-go/session-test", 80)
	_, cmd := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'k'}})
	if cmd == nil {
		t.Fatal("cmd = nil; want WorktreeExitMsg")
	}
	msg := cmd()
	wm, ok := msg.(WorktreeExitMsg)
	if !ok {
		t.Fatalf("cmd() = %T; want WorktreeExitMsg", msg)
	}
	if wm.Action != WorktreeActionKeep {
		t.Errorf("Action = %d; want WorktreeActionKeep", wm.Action)
	}
}

func TestWorktreeDialogModel_DiscardKey(t *testing.T) {
	m := NewWorktreeDialogModel("pi-go/session-test", 80)
	_, cmd := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'d'}})
	if cmd == nil {
		t.Fatal("cmd = nil; want WorktreeExitMsg")
	}
	msg := cmd()
	wm, ok := msg.(WorktreeExitMsg)
	if !ok {
		t.Fatalf("cmd() = %T; want WorktreeExitMsg", msg)
	}
	if wm.Action != WorktreeActionDiscard {
		t.Errorf("Action = %d; want WorktreeActionDiscard", wm.Action)
	}
}

func TestWorktreeDialogModel_EscCancels(t *testing.T) {
	m := NewWorktreeDialogModel("pi-go/session-test", 80)
	_, cmd := m.Update(tea.KeyMsg{Type: tea.KeyEsc})
	if cmd == nil {
		t.Fatal("cmd = nil; want DismissOverlayMsg")
	}
	msg := cmd()
	if _, ok := msg.(DismissOverlayMsg); !ok {
		t.Errorf("cmd() = %T; want DismissOverlayMsg", msg)
	}
}

func TestWorktreeDialogModel_View(t *testing.T) {
	m := NewWorktreeDialogModel("pi-go/session-test", 80)
	view := m.View()

	if !strings.Contains(view, "Session Worktree") {
		t.Error("View missing title 'Session Worktree'")
	}
	if !strings.Contains(view, "[m]") {
		t.Error("View missing merge option '[m]'")
	}
	if !strings.Contains(view, "[k]") {
		t.Error("View missing keep option '[k]'")
	}
	if !strings.Contains(view, "[d]") {
		t.Error("View missing discard option '[d]'")
	}
	if !strings.Contains(view, "session-test") {
		t.Error("View missing branch name")
	}
}
