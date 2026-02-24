// ABOUTME: Tests for TUI interaction improvements: queue editing, inline perm dialog, auto-accept, /compact fix
// ABOUTME: Covers arrow-up queue cycling, Shift+Tab auto-accept toggle, and slash command non-pollution

package btea

import (
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/mauromedda/pi-coding-agent-go/pkg/ai"
)

// --- Feature 1: Arrow-Up Queue Editing ---

func TestAppModel_ArrowUp_CyclesQueueWhenAgentRunning(t *testing.T) {
	m := NewAppModel(testDeps())
	m.agentRunning = true
	m.promptQueue = []string{"first", "second", "third"}
	m.editor = m.editor.SetText("current draft")

	// Arrow up should load last queued item
	key := tea.KeyMsg{Type: tea.KeyUp}
	result, _ := m.Update(key)
	model := result.(AppModel)

	if model.queueEditIndex != 0 {
		t.Errorf("queueEditIndex = %d; want 0", model.queueEditIndex)
	}
	if got := model.editor.Text(); got != "first" {
		t.Errorf("editor = %q; want %q", got, "first")
	}
	if model.savedDraft != "current draft" {
		t.Errorf("savedDraft = %q; want %q", model.savedDraft, "current draft")
	}
}

func TestAppModel_ArrowUp_CyclesThroughMultipleQueueItems(t *testing.T) {
	m := NewAppModel(testDeps())
	m.agentRunning = true
	m.promptQueue = []string{"first", "second", "third"}

	key := tea.KeyMsg{Type: tea.KeyUp}

	// First up: index 0 -> "first"
	result, _ := m.Update(key)
	m = result.(AppModel)
	if got := m.editor.Text(); got != "first" {
		t.Errorf("after 1st up: editor = %q; want %q", got, "first")
	}

	// Second up: index 1 -> "second"
	result, _ = m.Update(key)
	m = result.(AppModel)
	if got := m.editor.Text(); got != "second" {
		t.Errorf("after 2nd up: editor = %q; want %q", got, "second")
	}

	// Third up: index 2 -> "third"
	result, _ = m.Update(key)
	m = result.(AppModel)
	if got := m.editor.Text(); got != "third" {
		t.Errorf("after 3rd up: editor = %q; want %q", got, "third")
	}

	// Fourth up: should stay at last item (no wrap)
	result, _ = m.Update(key)
	m = result.(AppModel)
	if m.queueEditIndex != 2 {
		t.Errorf("queueEditIndex = %d; want 2 (clamped)", m.queueEditIndex)
	}
}

func TestAppModel_ArrowDown_RestoresDraftFromQueue(t *testing.T) {
	m := NewAppModel(testDeps())
	m.agentRunning = true
	m.promptQueue = []string{"first", "second"}
	m.editor = m.editor.SetText("my draft")

	upKey := tea.KeyMsg{Type: tea.KeyUp}
	downKey := tea.KeyMsg{Type: tea.KeyDown}

	// Go up twice
	result, _ := m.Update(upKey)
	m = result.(AppModel)
	result, _ = m.Update(upKey)
	m = result.(AppModel)

	// Go down once: back to "first"
	result, _ = m.Update(downKey)
	m = result.(AppModel)
	if got := m.editor.Text(); got != "first" {
		t.Errorf("after down: editor = %q; want %q", got, "first")
	}

	// Go down again: restore draft
	result, _ = m.Update(downKey)
	m = result.(AppModel)
	if got := m.editor.Text(); got != "my draft" {
		t.Errorf("after restoring draft: editor = %q; want %q", got, "my draft")
	}
	if m.queueEditIndex != -1 {
		t.Errorf("queueEditIndex = %d; want -1", m.queueEditIndex)
	}
}

func TestAppModel_EnterWhileEditingQueue_ReplacesItem(t *testing.T) {
	m := NewAppModel(testDeps())
	m.agentRunning = true
	m.promptQueue = []string{"original prompt", "other"}

	// Arrow up to select first queue item
	result, _ := m.Update(tea.KeyMsg{Type: tea.KeyUp})
	m = result.(AppModel)

	// Edit text in editor
	m.editor = m.editor.SetText("edited prompt")

	// Press enter: should replace queue item, not enqueue new
	result, _ = m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	m = result.(AppModel)

	if len(m.promptQueue) != 2 {
		t.Fatalf("promptQueue length = %d; want 2", len(m.promptQueue))
	}
	if m.promptQueue[0] != "edited prompt" {
		t.Errorf("promptQueue[0] = %q; want %q", m.promptQueue[0], "edited prompt")
	}
	if m.queueEditIndex != -1 {
		t.Errorf("queueEditIndex = %d; want -1 after enter", m.queueEditIndex)
	}
	if m.editor.Text() != "" {
		t.Errorf("editor should be cleared after queue edit submit")
	}
}

func TestAppModel_EnterWhileEditingQueue_AgentStopped_DrainResumes(t *testing.T) {
	m := NewAppModel(testDeps())
	m.agentRunning = true
	m.promptQueue = []string{"original", "second"}

	// Arrow up to edit first queue item
	result, _ := m.Update(tea.KeyMsg{Type: tea.KeyUp})
	m = result.(AppModel)

	// Agent finishes while user is editing
	m.agentRunning = false

	// User edits and presses Enter
	m.editor = m.editor.SetText("edited")
	result, cmd := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	m = result.(AppModel)

	// Queue item should be updated
	// Since agent stopped, drain should resume: "edited" replaces index 0,
	// then the drain pops "edited" from queue and submits it
	if m.queueEditIndex != -1 {
		t.Errorf("queueEditIndex = %d; want -1", m.queueEditIndex)
	}
	// The drain should have submitted the first item
	if cmd == nil {
		t.Error("expected a cmd from drain resuming")
	}
}

func TestAppModel_AgentDone_SkipsDrainDuringQueueEdit(t *testing.T) {
	m := NewAppModel(testDeps())
	m.agentRunning = true
	m.promptQueue = []string{"editing this", "other"}
	m.queueEditIndex = 0
	m.savedDraft = ""

	// Agent finishes: should NOT drain because user is editing
	result, _ := m.Update(AgentDoneMsg{})
	model := result.(AppModel)

	if len(model.promptQueue) != 2 {
		t.Errorf("promptQueue length = %d; want 2 (drain should be skipped)", len(model.promptQueue))
	}
}

func TestAppModel_ArrowUp_NoopWhenQueueEmpty(t *testing.T) {
	m := NewAppModel(testDeps())
	m.agentRunning = true
	m.promptQueue = nil
	m.editor = m.editor.SetText("some text")

	result, _ := m.Update(tea.KeyMsg{Type: tea.KeyUp})
	model := result.(AppModel)

	// Should not enter queue edit mode
	if model.queueEditIndex != -1 {
		t.Errorf("queueEditIndex = %d; want -1 when queue empty", model.queueEditIndex)
	}
}

// --- Feature 2: Inline Permission Dialog ---

func TestPermDialogModel_IsInlineOverlay(t *testing.T) {
	ch := make(chan PermissionReply, 1)
	pd := NewPermDialogModel("bash", nil, ch)

	if !isInlineOverlay(pd) {
		t.Error("PermDialogModel should be an inline overlay")
	}
	if isDropdownOverlay(pd) {
		t.Error("PermDialogModel should NOT be a dropdown overlay")
	}
}

func TestPermDialogModel_CompactView(t *testing.T) {
	ch := make(chan PermissionReply, 1)
	pd := NewPermDialogModel("bash", map[string]any{"cmd": "ls"}, ch)
	pd.width = 80

	view := pd.View()

	// Should contain tool name and keybinding hints
	if !strings.Contains(view, "bash") {
		t.Error("view should contain tool name")
	}
	if !strings.Contains(view, "[y]") {
		t.Error("view should contain [y] hint")
	}
	if !strings.Contains(view, "[a]") {
		t.Error("view should contain [a] hint")
	}
	if !strings.Contains(view, "[n]") {
		t.Error("view should contain [n] hint")
	}
}

func TestAppModel_PermDialog_RendersInline(t *testing.T) {
	m := NewAppModel(testDeps())
	m.width = 80
	m.height = 40
	m = m.propagateSize(tea.WindowSizeMsg{Width: 80, Height: 40})

	ch := make(chan PermissionReply, 1)
	m.overlay = NewPermDialogModel("bash", nil, ch)

	view := m.View()

	// The perm dialog should render inline (between content and editor),
	// not via overlayRender. Check that the editor prompt appears AFTER the dialog.
	permIdx := strings.Index(view, "[y]")
	editorIdx := strings.Index(view, "❯")

	if permIdx < 0 {
		t.Fatal("permission dialog hints not found in View()")
	}
	if editorIdx < 0 {
		t.Fatal("editor prompt not found in View()")
	}
	if permIdx > editorIdx {
		t.Errorf("perm dialog (pos %d) should render before editor prompt (pos %d)", permIdx, editorIdx)
	}
}

// --- Feature 3: Shift+Tab Auto-Accept Mode ---

func TestAppModel_ShiftTab_TogglesAutoAccept(t *testing.T) {
	m := NewAppModel(testDeps())

	key := tea.KeyMsg{Type: tea.KeyShiftTab}

	// First toggle: enable auto-accept
	result, _ := m.Update(key)
	model := result.(AppModel)
	if !model.autoAccept {
		t.Error("autoAccept = false; want true after first Shift+Tab")
	}

	// Second toggle: disable auto-accept
	result, _ = model.Update(key)
	model = result.(AppModel)
	if model.autoAccept {
		t.Error("autoAccept = true; want false after second Shift+Tab")
	}
}

func TestAppModel_AutoAccept_SkipsPermDialog(t *testing.T) {
	m := NewAppModel(testDeps())
	m.autoAccept = true

	ch := make(chan PermissionReply, 1)
	msg := PermissionRequestMsg{
		Tool:    "bash",
		Args:    map[string]any{"cmd": "ls"},
		ReplyCh: ch,
	}

	result, _ := m.Update(msg)
	model := result.(AppModel)

	// Should NOT show overlay
	if model.overlay != nil {
		t.Errorf("overlay = %T; want nil when autoAccept is enabled", model.overlay)
	}

	// Should have auto-replied
	select {
	case reply := <-ch:
		if !reply.Allowed {
			t.Error("reply.Allowed = false; want true (auto-accept)")
		}
	default:
		t.Error("no reply sent on channel; auto-accept should reply immediately")
	}
}

func TestAppModel_AutoAccept_FooterIndicator(t *testing.T) {
	m := NewAppModel(testDeps())
	m.autoAccept = true
	m.footer = m.footer.WithAutoAccept(true)
	m.width = 80

	view := m.footer.View()
	if !strings.Contains(view, "auto") {
		t.Error("footer should show auto-accept indicator")
	}
}

func TestAppModel_AltP_TogglesMode(t *testing.T) {
	m := NewAppModel(testDeps())

	// Alt+p should toggle Plan/Edit mode (moved from Shift+Tab)
	key := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'p'}, Alt: true}
	result, _ := m.Update(key)
	model := result.(AppModel)

	if model.mode != ModePlan {
		t.Errorf("after alt+p: mode = %v; want ModePlan", model.mode)
	}

	// Toggle back
	result, _ = model.Update(key)
	model = result.(AppModel)
	if model.mode != ModeEdit {
		t.Errorf("after second alt+p: mode = %v; want ModeEdit", model.mode)
	}
}

// --- Feature 4: Fix /compact ---

func TestAppModel_SlashCommand_DoesNotPolluteMessages(t *testing.T) {
	m := NewAppModel(testDeps())
	m.editor = m.editor.SetText("/help")

	key := tea.KeyMsg{Type: tea.KeyEnter}
	result, _ := m.Update(key)
	model := result.(AppModel)

	// Slash commands should NOT be added to m.messages (AI conversation)
	for _, msg := range model.messages {
		for _, c := range msg.Content {
			if c.Type == "text" && strings.Contains(c.Text, "/help") {
				t.Error("slash command text should not pollute AI messages")
			}
		}
	}
}

func TestAppModel_SlashCommand_DoesNotAddUserMsgContent(t *testing.T) {
	m := NewAppModel(testDeps())
	m.editor = m.editor.SetText("/help")

	key := tea.KeyMsg{Type: tea.KeyEnter}
	result, _ := m.Update(key)
	model := result.(AppModel)

	// Should NOT have a UserMsgModel in content for slash commands
	for _, c := range model.content {
		if _, ok := c.(UserMsgModel); ok {
			t.Error("slash commands should not add UserMsgModel to content")
		}
	}
}

func TestAppModel_BashCommand_StillAddsToContent(t *testing.T) {
	m := NewAppModel(testDeps())
	m.editor = m.editor.SetText("!ls")

	key := tea.KeyMsg{Type: tea.KeyEnter}
	result, _ := m.Update(key)
	model := result.(AppModel)

	// Bash commands SHOULD still add UserMsgModel (they're displayed to user)
	found := false
	for _, c := range model.content {
		if _, ok := c.(UserMsgModel); ok {
			found = true
			break
		}
	}
	if !found {
		t.Error("bash commands should still add UserMsgModel to content")
	}
}

func TestAppModel_NormalPrompt_StillAddsToMessages(t *testing.T) {
	m := NewAppModel(testDeps())
	m.editor = m.editor.SetText("hello world")

	key := tea.KeyMsg{Type: tea.KeyEnter}
	result, _ := m.Update(key)
	model := result.(AppModel)

	if len(model.messages) == 0 {
		t.Fatal("normal prompts should be added to messages")
	}
}

func TestAppModel_CompactDone_ShowsFeedback(t *testing.T) {
	m := NewAppModel(testDeps())
	m.compacting = true
	m.width = 80
	m.messages = []ai.Message{
		ai.NewTextMessage(ai.RoleUser, "hello"),
	}

	compacted := []ai.Message{
		ai.NewTextMessage(ai.RoleUser, "[Summary]"),
	}

	result, _ := m.Update(CompactDoneMsg{
		Messages:    compacted,
		Summary:     "Summarized",
		TokensSaved: 500,
	})
	model := result.(AppModel)

	// Should have added feedback to content
	found := false
	for _, c := range model.content {
		if am, ok := c.(*AssistantMsgModel); ok {
			text := am.Text()
			if strings.Contains(text, "500") || strings.Contains(strings.ToLower(text), "compact") {
				found = true
				break
			}
		}
	}
	if !found {
		t.Error("CompactDoneMsg should add visible feedback to content")
	}
}
