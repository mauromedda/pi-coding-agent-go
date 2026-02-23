// ABOUTME: Tests for prompt queue, history navigation, and type-ahead enqueue
// ABOUTME: Covers queue drain on AgentDoneMsg, Up/Down history, and Ctrl+E overlay

package btea

import (
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/mauromedda/pi-coding-agent-go/pkg/ai"
)

// --- History navigation tests ---

func TestAppModel_HistoryPopulatedOnSubmit(t *testing.T) {
	m := NewAppModel(testDeps())
	m.editor = m.editor.SetText("first prompt")

	result, _ := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	m = result.(AppModel)

	if len(m.promptHistory) != 1 {
		t.Fatalf("promptHistory length = %d; want 1", len(m.promptHistory))
	}
	if m.promptHistory[0] != "first prompt" {
		t.Errorf("promptHistory[0] = %q; want %q", m.promptHistory[0], "first prompt")
	}
	if m.historyIndex != -1 {
		t.Errorf("historyIndex = %d; want -1", m.historyIndex)
	}
}

func TestAppModel_HistoryUpCyclesPrevious(t *testing.T) {
	m := NewAppModel(testDeps())

	// Submit two prompts (directly populate history to avoid agent start)
	m.promptHistory = []string{"first", "second"}
	m.historyIndex = -1

	// Press Up: should show "second" (most recent)
	result, _ := m.Update(tea.KeyMsg{Type: tea.KeyUp})
	m = result.(AppModel)

	if got := m.editor.Text(); got != "second" {
		t.Errorf("after first Up: editor = %q; want %q", got, "second")
	}
	if m.historyIndex != 0 {
		t.Errorf("historyIndex = %d; want 0", m.historyIndex)
	}

	// Press Up again: should show "first"
	result, _ = m.Update(tea.KeyMsg{Type: tea.KeyUp})
	m = result.(AppModel)

	if got := m.editor.Text(); got != "first" {
		t.Errorf("after second Up: editor = %q; want %q", got, "first")
	}
	if m.historyIndex != 1 {
		t.Errorf("historyIndex = %d; want 1", m.historyIndex)
	}

	// Press Up at oldest: should stay at "first"
	result, _ = m.Update(tea.KeyMsg{Type: tea.KeyUp})
	m = result.(AppModel)

	if got := m.editor.Text(); got != "first" {
		t.Errorf("after third Up (at oldest): editor = %q; want %q", got, "first")
	}
}

func TestAppModel_HistoryDownReturns(t *testing.T) {
	m := NewAppModel(testDeps())
	m.promptHistory = []string{"first", "second"}
	m.historyIndex = -1

	// Navigate up twice
	result, _ := m.Update(tea.KeyMsg{Type: tea.KeyUp})
	m = result.(AppModel)
	result, _ = m.Update(tea.KeyMsg{Type: tea.KeyUp})
	m = result.(AppModel)

	// Press Down: should show "second"
	result, _ = m.Update(tea.KeyMsg{Type: tea.KeyUp}) // at "first", historyIndex=1
	m = result.(AppModel)

	// Navigate back with Down
	m.historyIndex = 1 // at "first"
	result, _ = m.Update(tea.KeyMsg{Type: tea.KeyDown})
	m = result.(AppModel)

	if got := m.editor.Text(); got != "second" {
		t.Errorf("after Down: editor = %q; want %q", got, "second")
	}
	if m.historyIndex != 0 {
		t.Errorf("historyIndex = %d; want 0", m.historyIndex)
	}
}

func TestAppModel_HistoryDownRestoresDraft(t *testing.T) {
	m := NewAppModel(testDeps())
	m.promptHistory = []string{"old prompt"}
	m.historyIndex = -1

	// Type a draft
	m.editor = m.editor.SetText("my draft")

	// Press Up: should save draft and show "old prompt"
	result, _ := m.Update(tea.KeyMsg{Type: tea.KeyUp})
	m = result.(AppModel)

	if m.savedDraft != "my draft" {
		t.Errorf("savedDraft = %q; want %q", m.savedDraft, "my draft")
	}
	if got := m.editor.Text(); got != "old prompt" {
		t.Errorf("editor = %q; want %q", got, "old prompt")
	}

	// Press Down: should restore draft
	result, _ = m.Update(tea.KeyMsg{Type: tea.KeyDown})
	m = result.(AppModel)

	if got := m.editor.Text(); got != "my draft" {
		t.Errorf("after Down: editor = %q; want %q", got, "my draft")
	}
	if m.historyIndex != -1 {
		t.Errorf("historyIndex = %d; want -1", m.historyIndex)
	}
}

func TestAppModel_HistoryUpIgnoredWhenAgentRunning(t *testing.T) {
	m := NewAppModel(testDeps())
	m.agentRunning = true
	m.promptHistory = []string{"old prompt"}
	m.editor = m.editor.SetText("current")

	// Up while agent running: should NOT navigate history
	result, _ := m.Update(tea.KeyMsg{Type: tea.KeyUp})
	m = result.(AppModel)

	// historyIndex should remain -1
	if m.historyIndex != -1 {
		t.Errorf("historyIndex = %d; want -1 (no history nav while agent runs)", m.historyIndex)
	}
}

func TestAppModel_HistoryUpOnlyOnFirstLine(t *testing.T) {
	m := NewAppModel(testDeps())
	m.promptHistory = []string{"old prompt"}

	// Multi-line editor: cursor on second line
	m.editor = m.editor.SetText("line1\nline2")
	// Cursor is at (1, 5) after SetText

	// Up should move cursor up in editor, NOT navigate history
	result, _ := m.Update(tea.KeyMsg{Type: tea.KeyUp})
	m = result.(AppModel)

	if m.historyIndex != -1 {
		t.Errorf("historyIndex = %d; want -1 (should not trigger on non-first line)", m.historyIndex)
	}
}

func TestAppModel_HistoryUpEmptyHistoryNoOp(t *testing.T) {
	m := NewAppModel(testDeps())
	m.editor = m.editor.SetText("current")

	// No history: Up should be a no-op
	result, _ := m.Update(tea.KeyMsg{Type: tea.KeyUp})
	m = result.(AppModel)

	if got := m.editor.Text(); got != "current" {
		t.Errorf("editor = %q; want %q (empty history, no change)", got, "current")
	}
}

// --- Type-ahead queue tests ---

func TestAppModel_EnqueueWhileAgentRunning(t *testing.T) {
	m := NewAppModel(testDeps())
	m.agentRunning = true
	m.editor = m.editor.SetText("queued prompt")

	result, cmd := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	m = result.(AppModel)

	if cmd != nil {
		t.Errorf("cmd = %v; want nil (enqueue should not start agent)", cmd)
	}
	if len(m.promptQueue) != 1 {
		t.Fatalf("promptQueue length = %d; want 1", len(m.promptQueue))
	}
	if m.promptQueue[0] != "queued prompt" {
		t.Errorf("promptQueue[0] = %q; want %q", m.promptQueue[0], "queued prompt")
	}
	if m.editor.Text() != "" {
		t.Errorf("editor should be cleared after enqueue, got %q", m.editor.Text())
	}
	if m.footer.queuedCount != 1 {
		t.Errorf("footer.queuedCount = %d; want 1", m.footer.queuedCount)
	}
}

func TestAppModel_EnqueueMultiplePrompts(t *testing.T) {
	m := NewAppModel(testDeps())
	m.agentRunning = true

	prompts := []string{"first", "second", "third"}
	for _, p := range prompts {
		m.editor = m.editor.SetText(p)
		result, _ := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
		m = result.(AppModel)
	}

	if len(m.promptQueue) != 3 {
		t.Fatalf("promptQueue length = %d; want 3", len(m.promptQueue))
	}
	for i, want := range prompts {
		if m.promptQueue[i] != want {
			t.Errorf("promptQueue[%d] = %q; want %q", i, m.promptQueue[i], want)
		}
	}
	if m.footer.queuedCount != 3 {
		t.Errorf("footer.queuedCount = %d; want 3", m.footer.queuedCount)
	}
}

func TestAppModel_EnqueueEmptyIgnored(t *testing.T) {
	m := NewAppModel(testDeps())
	m.agentRunning = true
	// Editor is empty

	result, _ := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	m = result.(AppModel)

	if len(m.promptQueue) != 0 {
		t.Errorf("promptQueue length = %d; want 0 (empty prompt ignored)", len(m.promptQueue))
	}
}

func TestAppModel_QueueDrainOnAgentDone(t *testing.T) {
	m := NewAppModel(testDeps())
	m.agentRunning = true
	m.promptQueue = []string{"next prompt"}
	m.footer = m.footer.WithQueuedCount(1)

	result, cmd := m.Update(AgentDoneMsg{Messages: []ai.Message{
		ai.NewTextMessage(ai.RoleUser, "first"),
	}})
	m = result.(AppModel)

	// Agent should be running again (drain triggered submitPrompt)
	if !m.agentRunning {
		t.Error("agentRunning = false; want true (queue drain should start next prompt)")
	}
	if cmd == nil {
		t.Error("cmd = nil; want non-nil (should start agent for drained prompt)")
	}
	if len(m.promptQueue) != 0 {
		t.Errorf("promptQueue length = %d; want 0 (drained)", len(m.promptQueue))
	}
	if m.footer.queuedCount != 0 {
		t.Errorf("footer.queuedCount = %d; want 0", m.footer.queuedCount)
	}
}

func TestAppModel_QueueDrainSequential(t *testing.T) {
	m := NewAppModel(testDeps())
	m.agentRunning = true
	m.promptQueue = []string{"second", "third"}
	m.footer = m.footer.WithQueuedCount(2)

	// First AgentDone: should drain "second"
	result, cmd := m.Update(AgentDoneMsg{})
	m = result.(AppModel)

	if len(m.promptQueue) != 1 {
		t.Fatalf("after first drain: promptQueue length = %d; want 1", len(m.promptQueue))
	}
	if m.promptQueue[0] != "third" {
		t.Errorf("promptQueue[0] = %q; want %q", m.promptQueue[0], "third")
	}
	if m.footer.queuedCount != 1 {
		t.Errorf("footer.queuedCount = %d; want 1", m.footer.queuedCount)
	}
	if cmd == nil {
		t.Error("cmd should not be nil (starting drained prompt)")
	}
}

func TestAppModel_QueueHistoryIntegration(t *testing.T) {
	m := NewAppModel(testDeps())
	m.agentRunning = true
	m.editor = m.editor.SetText("queued one")

	// Enqueue: history is NOT populated yet (only on actual submit via drain)
	result, _ := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	m = result.(AppModel)

	if len(m.promptHistory) != 0 {
		t.Errorf("promptHistory length = %d; want 0 (history populated on drain, not enqueue)", len(m.promptHistory))
	}

	// Drain: AgentDoneMsg triggers submitPrompt which adds to history
	result, _ = m.Update(AgentDoneMsg{})
	m = result.(AppModel)

	if len(m.promptHistory) != 1 {
		t.Fatalf("promptHistory length = %d; want 1 after drain", len(m.promptHistory))
	}
	if m.promptHistory[0] != "queued one" {
		t.Errorf("promptHistory[0] = %q; want %q", m.promptHistory[0], "queued one")
	}
}
