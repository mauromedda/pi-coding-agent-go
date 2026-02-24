// ABOUTME: Tests for BackgroundViewModel overlay: navigation, dismiss, cancel, review
// ABOUTME: Validates key handling and View rendering for background task list

package btea

import (
	"strings"
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"
)

func TestBackgroundViewModel_Navigation(t *testing.T) {
	tasks := []BackgroundTask{
		{ID: "bg-1", Prompt: "first", Status: BGRunning, StartedAt: time.Now()},
		{ID: "bg-2", Prompt: "second", Status: BGDone, StartedAt: time.Now()},
		{ID: "bg-3", Prompt: "third", Status: BGFailed, StartedAt: time.Now()},
	}
	m := NewBackgroundViewModel(tasks, 80, 24)

	if m.cursor != 0 {
		t.Errorf("initial cursor = %d; want 0", m.cursor)
	}

	// Move down
	result, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'j'}})
	m = result.(BackgroundViewModel)
	if m.cursor != 1 {
		t.Errorf("cursor after j = %d; want 1", m.cursor)
	}

	// Move down again
	result, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'j'}})
	m = result.(BackgroundViewModel)
	if m.cursor != 2 {
		t.Errorf("cursor after j = %d; want 2", m.cursor)
	}

	// Can't go past end
	result, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'j'}})
	m = result.(BackgroundViewModel)
	if m.cursor != 2 {
		t.Errorf("cursor after j at end = %d; want 2", m.cursor)
	}

	// Move up
	result, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'k'}})
	m = result.(BackgroundViewModel)
	if m.cursor != 1 {
		t.Errorf("cursor after k = %d; want 1", m.cursor)
	}
}

func TestBackgroundViewModel_DismissCompleted(t *testing.T) {
	tasks := []BackgroundTask{
		{ID: "bg-1", Prompt: "done task", Status: BGDone, StartedAt: time.Now()},
		{ID: "bg-2", Prompt: "running task", Status: BGRunning, StartedAt: time.Now()},
	}
	m := NewBackgroundViewModel(tasks, 80, 24)

	// Dismiss the done task
	result, cmd := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'d'}})
	m = result.(BackgroundViewModel)
	if len(m.tasks) != 1 {
		t.Fatalf("tasks len = %d; want 1 after dismissing done task", len(m.tasks))
	}
	if m.tasks[0].ID != "bg-2" {
		t.Errorf("remaining task ID = %q; want bg-2", m.tasks[0].ID)
	}
	// Verify it sends BackgroundTaskRemoveMsg to sync the manager
	if cmd == nil {
		t.Fatal("cmd = nil; want BackgroundTaskRemoveMsg")
	}
	msg := cmd()
	rmMsg, ok := msg.(BackgroundTaskRemoveMsg)
	if !ok {
		t.Fatalf("cmd() = %T; want BackgroundTaskRemoveMsg", msg)
	}
	if rmMsg.TaskID != "bg-1" {
		t.Errorf("TaskID = %q; want bg-1", rmMsg.TaskID)
	}
}

func TestBackgroundViewModel_CannotDismissRunning(t *testing.T) {
	tasks := []BackgroundTask{
		{ID: "bg-1", Prompt: "running task", Status: BGRunning, StartedAt: time.Now()},
	}
	m := NewBackgroundViewModel(tasks, 80, 24)

	result, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'d'}})
	m = result.(BackgroundViewModel)
	if len(m.tasks) != 1 {
		t.Error("should not dismiss running task")
	}
}

func TestBackgroundViewModel_CancelRunning(t *testing.T) {
	tasks := []BackgroundTask{
		{ID: "bg-1", Prompt: "running task", Status: BGRunning, StartedAt: time.Now()},
	}
	m := NewBackgroundViewModel(tasks, 80, 24)

	_, cmd := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'c'}})
	if cmd == nil {
		t.Fatal("cmd = nil; want BackgroundTaskCancelMsg")
	}
	msg := cmd()
	cancelMsg, ok := msg.(BackgroundTaskCancelMsg)
	if !ok {
		t.Fatalf("cmd() = %T; want BackgroundTaskCancelMsg", msg)
	}
	if cancelMsg.TaskID != "bg-1" {
		t.Errorf("TaskID = %q; want bg-1", cancelMsg.TaskID)
	}
}

func TestBackgroundViewModel_ReviewCompleted(t *testing.T) {
	tasks := []BackgroundTask{
		{ID: "bg-review", Prompt: "done", Status: BGDone, StartedAt: time.Now()},
	}
	m := NewBackgroundViewModel(tasks, 80, 24)

	_, cmd := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	if cmd == nil {
		t.Fatal("cmd = nil; want BackgroundTaskReviewMsg cmd")
	}
	msg := cmd()
	review, ok := msg.(BackgroundTaskReviewMsg)
	if !ok {
		t.Fatalf("cmd() = %T; want BackgroundTaskReviewMsg", msg)
	}
	if review.TaskID != "bg-review" {
		t.Errorf("TaskID = %q; want %q", review.TaskID, "bg-review")
	}
}

func TestBackgroundViewModel_EscCloses(t *testing.T) {
	m := NewBackgroundViewModel(nil, 80, 24)
	_, cmd := m.Update(tea.KeyMsg{Type: tea.KeyEsc})
	if cmd == nil {
		t.Fatal("cmd = nil; want DismissOverlayMsg cmd")
	}
	msg := cmd()
	if _, ok := msg.(DismissOverlayMsg); !ok {
		t.Errorf("cmd() = %T; want DismissOverlayMsg", msg)
	}
}

func TestBackgroundViewModel_View(t *testing.T) {
	tasks := []BackgroundTask{
		{ID: "bg-1", Prompt: "hello world", Status: BGRunning, StartedAt: time.Now()},
		{ID: "bg-2", Prompt: "done thing", Status: BGDone, StartedAt: time.Now()},
	}
	m := NewBackgroundViewModel(tasks, 80, 24)
	view := m.View()

	if !strings.Contains(view, "Background Tasks") {
		t.Error("View missing title 'Background Tasks'")
	}
	if !strings.Contains(view, "bg-1") {
		t.Error("View missing task ID 'bg-1'")
	}
	if !strings.Contains(view, "bg-2") {
		t.Error("View missing task ID 'bg-2'")
	}
	if !strings.Contains(view, "⠋") {
		t.Error("View missing running icon '⠋'")
	}
	if !strings.Contains(view, "✓") {
		t.Error("View missing done icon '✓'")
	}
}

func TestBackgroundViewModel_EmptyView(t *testing.T) {
	m := NewBackgroundViewModel(nil, 80, 24)
	view := m.View()

	if !strings.Contains(view, "no background tasks") {
		t.Error("View missing empty message")
	}
}
