// ABOUTME: Tests for QueueViewModel overlay: navigation, delete, reorder, edit, close
// ABOUTME: Verifies vim-style key bindings and message emission for queue management

package btea

import (
	"testing"

	tea "github.com/charmbracelet/bubbletea"
)

// Compile-time check: QueueViewModel must satisfy tea.Model.
var _ tea.Model = QueueViewModel{}

func TestQueueViewModel_Init(t *testing.T) {
	m := NewQueueViewModel([]string{"a", "b"}, 80)
	if cmd := m.Init(); cmd != nil {
		t.Errorf("Init() returned non-nil cmd")
	}
}

func TestQueueViewModel_CursorNavigation(t *testing.T) {
	m := NewQueueViewModel([]string{"a", "b", "c"}, 80)

	if m.cursor != 0 {
		t.Fatalf("initial cursor = %d; want 0", m.cursor)
	}

	// Move down with j
	result, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'j'}})
	m = result.(QueueViewModel)
	if m.cursor != 1 {
		t.Errorf("after j: cursor = %d; want 1", m.cursor)
	}

	// Move down with down arrow
	result, _ = m.Update(tea.KeyMsg{Type: tea.KeyDown})
	m = result.(QueueViewModel)
	if m.cursor != 2 {
		t.Errorf("after down: cursor = %d; want 2", m.cursor)
	}

	// Down at bottom: should not go past end
	result, _ = m.Update(tea.KeyMsg{Type: tea.KeyDown})
	m = result.(QueueViewModel)
	if m.cursor != 2 {
		t.Errorf("down at bottom: cursor = %d; want 2", m.cursor)
	}

	// Move up with k
	result, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'k'}})
	m = result.(QueueViewModel)
	if m.cursor != 1 {
		t.Errorf("after k: cursor = %d; want 1", m.cursor)
	}

	// Move up with up arrow
	result, _ = m.Update(tea.KeyMsg{Type: tea.KeyUp})
	m = result.(QueueViewModel)
	if m.cursor != 0 {
		t.Errorf("after up: cursor = %d; want 0", m.cursor)
	}

	// Up at top: should stay
	result, _ = m.Update(tea.KeyMsg{Type: tea.KeyUp})
	m = result.(QueueViewModel)
	if m.cursor != 0 {
		t.Errorf("up at top: cursor = %d; want 0", m.cursor)
	}
}

func TestQueueViewModel_DeleteItem(t *testing.T) {
	m := NewQueueViewModel([]string{"a", "b", "c"}, 80)

	// Delete first item with 'd'
	result, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'d'}})
	m = result.(QueueViewModel)

	if len(m.items) != 2 {
		t.Fatalf("items length = %d; want 2", len(m.items))
	}
	if m.items[0] != "b" || m.items[1] != "c" {
		t.Errorf("items = %v; want [b c]", m.items)
	}
}

func TestQueueViewModel_DeleteWithBackspace(t *testing.T) {
	m := NewQueueViewModel([]string{"a", "b"}, 80)

	result, _ := m.Update(tea.KeyMsg{Type: tea.KeyBackspace})
	m = result.(QueueViewModel)

	if len(m.items) != 1 {
		t.Fatalf("items length = %d; want 1", len(m.items))
	}
	if m.items[0] != "b" {
		t.Errorf("items[0] = %q; want %q", m.items[0], "b")
	}
}

func TestQueueViewModel_DeleteClampsCursor(t *testing.T) {
	m := NewQueueViewModel([]string{"a", "b"}, 80)
	m.cursor = 1 // at "b"

	// Delete: removes "b", cursor should clamp to 0
	result, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'d'}})
	m = result.(QueueViewModel)

	if m.cursor != 0 {
		t.Errorf("cursor = %d; want 0 after deleting last item", m.cursor)
	}
}

func TestQueueViewModel_DeleteLastItemCloses(t *testing.T) {
	m := NewQueueViewModel([]string{"only"}, 80)

	_, cmd := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'d'}})

	if cmd == nil {
		t.Fatal("cmd = nil; want QueueUpdatedMsg when last item deleted")
	}
	msg := cmd()
	qum, ok := msg.(QueueUpdatedMsg)
	if !ok {
		t.Fatalf("cmd() = %T; want QueueUpdatedMsg", msg)
	}
	if len(qum.Items) != 0 {
		t.Errorf("QueueUpdatedMsg.Items length = %d; want 0", len(qum.Items))
	}
}

func TestQueueViewModel_SwapDown(t *testing.T) {
	m := NewQueueViewModel([]string{"a", "b", "c"}, 80)
	m.cursor = 0

	// Shift+J: swap down
	result, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'J'}})
	m = result.(QueueViewModel)

	if m.items[0] != "b" || m.items[1] != "a" {
		t.Errorf("after swap down: items = %v; want [b a c]", m.items)
	}
	if m.cursor != 1 {
		t.Errorf("cursor = %d; want 1 (follows swapped item)", m.cursor)
	}
}

func TestQueueViewModel_SwapUp(t *testing.T) {
	m := NewQueueViewModel([]string{"a", "b", "c"}, 80)
	m.cursor = 2

	// Shift+K: swap up
	result, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'K'}})
	m = result.(QueueViewModel)

	if m.items[1] != "c" || m.items[2] != "b" {
		t.Errorf("after swap up: items = %v; want [a c b]", m.items)
	}
	if m.cursor != 1 {
		t.Errorf("cursor = %d; want 1 (follows swapped item)", m.cursor)
	}
}

func TestQueueViewModel_SwapDownAtBottom(t *testing.T) {
	m := NewQueueViewModel([]string{"a", "b"}, 80)
	m.cursor = 1

	// Shift+J at bottom: no-op
	result, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'J'}})
	m = result.(QueueViewModel)

	if m.items[0] != "a" || m.items[1] != "b" {
		t.Errorf("swap at bottom should be no-op: items = %v", m.items)
	}
}

func TestQueueViewModel_SwapUpAtTop(t *testing.T) {
	m := NewQueueViewModel([]string{"a", "b"}, 80)
	m.cursor = 0

	// Shift+K at top: no-op
	result, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'K'}})
	m = result.(QueueViewModel)

	if m.items[0] != "a" || m.items[1] != "b" {
		t.Errorf("swap at top should be no-op: items = %v", m.items)
	}
}

func TestQueueViewModel_EditItem(t *testing.T) {
	m := NewQueueViewModel([]string{"first", "second"}, 80)
	m.cursor = 0

	// 'e' or Enter: edit item
	_, cmd := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'e'}})

	if cmd == nil {
		t.Fatal("cmd = nil; want QueueEditMsg")
	}
	msg := cmd()
	editMsg, ok := msg.(QueueEditMsg)
	if !ok {
		t.Fatalf("cmd() = %T; want QueueEditMsg", msg)
	}
	if editMsg.Text != "first" {
		t.Errorf("QueueEditMsg.Text = %q; want %q", editMsg.Text, "first")
	}
	if editMsg.Index != 0 {
		t.Errorf("QueueEditMsg.Index = %d; want 0", editMsg.Index)
	}
}

func TestQueueViewModel_EditWithEnter(t *testing.T) {
	m := NewQueueViewModel([]string{"first"}, 80)

	_, cmd := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	if cmd == nil {
		t.Fatal("cmd = nil; want QueueEditMsg")
	}
	msg := cmd()
	if _, ok := msg.(QueueEditMsg); !ok {
		t.Errorf("cmd() = %T; want QueueEditMsg", msg)
	}
}

func TestQueueViewModel_CloseWithEsc(t *testing.T) {
	m := NewQueueViewModel([]string{"a", "b"}, 80)

	_, cmd := m.Update(tea.KeyMsg{Type: tea.KeyEsc})
	if cmd == nil {
		t.Fatal("cmd = nil; want QueueUpdatedMsg")
	}
	msg := cmd()
	qum, ok := msg.(QueueUpdatedMsg)
	if !ok {
		t.Fatalf("cmd() = %T; want QueueUpdatedMsg", msg)
	}
	if len(qum.Items) != 2 {
		t.Errorf("QueueUpdatedMsg.Items length = %d; want 2", len(qum.Items))
	}
}

func TestQueueViewModel_CloseWithQ(t *testing.T) {
	m := NewQueueViewModel([]string{"a"}, 80)

	_, cmd := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'q'}})
	if cmd == nil {
		t.Fatal("cmd = nil; want QueueUpdatedMsg")
	}
	msg := cmd()
	if _, ok := msg.(QueueUpdatedMsg); !ok {
		t.Fatalf("cmd() = %T; want QueueUpdatedMsg", msg)
	}
}

func TestQueueViewModel_View(t *testing.T) {
	m := NewQueueViewModel([]string{"prompt one", "prompt two"}, 80)
	view := m.View()

	if view == "" {
		t.Error("View() returned empty string")
	}
}

// --- App-level Ctrl+E and message handling tests ---

func TestAppModel_CtrlEOpensQueueOverlay(t *testing.T) {
	m := NewAppModel(testDeps())
	m.promptQueue = []string{"queued prompt"}

	key := tea.KeyMsg{Type: tea.KeyCtrlE}
	result, _ := m.Update(key)
	model := result.(AppModel)

	if model.overlay == nil {
		t.Fatal("overlay = nil; want QueueViewModel")
	}
	if _, ok := model.overlay.(QueueViewModel); !ok {
		t.Errorf("overlay = %T; want QueueViewModel", model.overlay)
	}
}

func TestAppModel_CtrlENoOpWhenQueueEmpty(t *testing.T) {
	m := NewAppModel(testDeps())

	key := tea.KeyMsg{Type: tea.KeyCtrlE}
	result, _ := m.Update(key)
	model := result.(AppModel)

	if model.overlay != nil {
		t.Errorf("overlay = %v; want nil when queue is empty", model.overlay)
	}
}

func TestAppModel_QueueUpdatedMsg(t *testing.T) {
	m := NewAppModel(testDeps())
	m.agentRunning = true // agent still running: no drain on close
	m.promptQueue = []string{"old1", "old2"}
	m.overlay = NewQueueViewModel(m.promptQueue, 80)

	result, _ := m.Update(QueueUpdatedMsg{Items: []string{"reordered"}})
	model := result.(AppModel)

	if model.overlay != nil {
		t.Error("overlay should be nil after QueueUpdatedMsg")
	}
	if len(model.promptQueue) != 1 {
		t.Fatalf("promptQueue length = %d; want 1", len(model.promptQueue))
	}
	if model.promptQueue[0] != "reordered" {
		t.Errorf("promptQueue[0] = %q; want %q", model.promptQueue[0], "reordered")
	}
	if model.footer.queuedCount != 1 {
		t.Errorf("footer.queuedCount = %d; want 1", model.footer.queuedCount)
	}
}

func TestAppModel_AgentDoneSkipsDrainWhileOverlayOpen(t *testing.T) {
	m := NewAppModel(testDeps())
	m.agentRunning = true
	m.promptQueue = []string{"next"}
	m.overlay = NewQueueViewModel(m.promptQueue, 80)

	// AgentDone while overlay is open: should NOT drain
	result, cmd := m.Update(AgentDoneMsg{})
	model := result.(AppModel)

	if model.agentRunning {
		t.Error("agentRunning should be false after AgentDoneMsg")
	}
	if cmd != nil {
		t.Error("cmd should be nil (drain skipped while overlay open)")
	}
	if len(model.promptQueue) != 1 {
		t.Errorf("promptQueue length = %d; want 1 (not drained)", len(model.promptQueue))
	}
}

func TestAppModel_QueueUpdatedResumeDrain(t *testing.T) {
	m := NewAppModel(testDeps())
	m.agentRunning = false // agent already finished
	m.promptQueue = []string{"stale"}
	m.overlay = NewQueueViewModel([]string{"queued"}, 80)

	// QueueUpdatedMsg with items while agent is idle: should drain
	result, cmd := m.Update(QueueUpdatedMsg{Items: []string{"drain me"}})
	model := result.(AppModel)

	if model.overlay != nil {
		t.Error("overlay should be nil after QueueUpdatedMsg")
	}
	if !model.agentRunning {
		t.Error("agentRunning should be true (drain resumed)")
	}
	if cmd == nil {
		t.Error("cmd should not be nil (starting drained prompt)")
	}
	if len(model.promptQueue) != 0 {
		t.Errorf("promptQueue length = %d; want 0 (drained)", len(model.promptQueue))
	}
}

func TestAppModel_QueueEditMsg(t *testing.T) {
	m := NewAppModel(testDeps())
	m.promptQueue = []string{"edit me", "keep me"}
	m.overlay = NewQueueViewModel(m.promptQueue, 80)

	result, _ := m.Update(QueueEditMsg{Text: "edit me", Index: 0})
	model := result.(AppModel)

	if model.overlay != nil {
		t.Error("overlay should be nil after QueueEditMsg")
	}
	// Item removed from queue
	if len(model.promptQueue) != 1 {
		t.Fatalf("promptQueue length = %d; want 1", len(model.promptQueue))
	}
	if model.promptQueue[0] != "keep me" {
		t.Errorf("promptQueue[0] = %q; want %q", model.promptQueue[0], "keep me")
	}
	// Editor should have the text
	if got := model.editor.Text(); got != "edit me" {
		t.Errorf("editor text = %q; want %q", got, "edit me")
	}
	if model.footer.queuedCount != 1 {
		t.Errorf("footer.queuedCount = %d; want 1", model.footer.queuedCount)
	}
}
