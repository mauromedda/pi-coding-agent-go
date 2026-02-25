// ABOUTME: Tests for submitOrEnqueue, shift+enter, and alt+enter key handling
// ABOUTME: Verifies submit/enqueue extraction and alternative keybindings

package btea

import (
	"testing"
)

// --- submitOrEnqueue tests ---

func TestAppModel_SubmitOrEnqueue_SubmitsWhenIdle(t *testing.T) {
	m := NewAppModel(testDeps())
	m.editor = m.editor.SetText("hello world")

	result, cmd := m.submitOrEnqueue()
	model := result

	if cmd == nil {
		t.Error("cmd = nil; want non-nil (should start agent)")
	}
	if model.editor.Text() != "" {
		t.Errorf("editor should be cleared; got %q", model.editor.Text())
	}
	if len(model.promptHistory) != 1 || model.promptHistory[0] != "hello world" {
		t.Errorf("promptHistory = %v; want [hello world]", model.promptHistory)
	}
}

func TestAppModel_SubmitOrEnqueue_EnqueuesWhenAgentRunning(t *testing.T) {
	m := NewAppModel(testDeps())
	m.agentRunning = true
	m.editor = m.editor.SetText("queued msg")

	result, cmd := m.submitOrEnqueue()
	model := result

	if cmd != nil {
		t.Errorf("cmd = %v; want nil (should enqueue, not start agent)", cmd)
	}
	if len(model.promptQueue) != 1 || model.promptQueue[0] != "queued msg" {
		t.Errorf("promptQueue = %v; want [queued msg]", model.promptQueue)
	}
	if model.editor.Text() != "" {
		t.Errorf("editor should be cleared; got %q", model.editor.Text())
	}
	if model.footer.queuedCount != 1 {
		t.Errorf("footer.queuedCount = %d; want 1", model.footer.queuedCount)
	}
}

func TestAppModel_SubmitOrEnqueue_QueueEditFinishWhileAgentRunning(t *testing.T) {
	m := NewAppModel(testDeps())
	m.agentRunning = true
	m.promptQueue = []string{"original"}
	m.queueEditIndex = 0
	m.editor = m.editor.SetText("edited")

	result, cmd := m.submitOrEnqueue()
	model := result

	// Agent still running: edit is saved, no drain occurs
	if cmd != nil {
		t.Errorf("cmd = %v; want nil", cmd)
	}
	if model.promptQueue[0] != "edited" {
		t.Errorf("promptQueue[0] = %q; want %q", model.promptQueue[0], "edited")
	}
	if model.queueEditIndex != -1 {
		t.Errorf("queueEditIndex = %d; want -1", model.queueEditIndex)
	}
}

func TestAppModel_SubmitOrEnqueue_QueueEditDrainsWhenIdle(t *testing.T) {
	m := NewAppModel(testDeps())
	m.agentRunning = false
	m.promptQueue = []string{"original"}
	m.queueEditIndex = 0
	m.editor = m.editor.SetText("edited")

	result, cmd := m.submitOrEnqueue()
	model := result

	// Agent idle with queued items: edit is saved, then drain submits it
	if cmd == nil {
		t.Error("cmd = nil; want non-nil (should drain and submit)")
	}
	if model.queueEditIndex != -1 {
		t.Errorf("queueEditIndex = %d; want -1", model.queueEditIndex)
	}
	// Queue should be drained (the edited item was submitted)
	if len(model.promptQueue) != 0 {
		t.Errorf("promptQueue length = %d; want 0 (drained)", len(model.promptQueue))
	}
}

func TestAppModel_SubmitOrEnqueue_EmptyEditorNoOp(t *testing.T) {
	m := NewAppModel(testDeps())
	// editor is empty

	result, _ := m.submitOrEnqueue()
	model := result

	if len(model.promptQueue) != 0 {
		t.Errorf("promptQueue should be empty; got %v", model.promptQueue)
	}
	if len(model.promptHistory) != 0 {
		t.Errorf("promptHistory should be empty; got %v", model.promptHistory)
	}
}

// --- Shift+Enter tests ---

func TestAppModel_ShiftEnter_SubmitsWhenIdle(t *testing.T) {
	// shift+enter delegates to submitOrEnqueue, same as enter.
	// We verify the extracted method works for idle submission.
	m := NewAppModel(testDeps())
	m.editor = m.editor.SetText("via shift enter")
	resultModel, resultCmd := m.submitOrEnqueue()

	if resultCmd == nil {
		t.Error("cmd = nil; want non-nil (shift+enter should submit)")
	}
	if resultModel.editor.Text() != "" {
		t.Errorf("editor should be cleared; got %q", resultModel.editor.Text())
	}
	if len(resultModel.promptHistory) != 1 || resultModel.promptHistory[0] != "via shift enter" {
		t.Errorf("promptHistory = %v; want [via shift enter]", resultModel.promptHistory)
	}
}

func TestAppModel_ShiftEnter_EnqueuesWhenAgentRunning(t *testing.T) {
	m := NewAppModel(testDeps())
	m.agentRunning = true
	m.editor = m.editor.SetText("shift queued")

	// Both shift+enter and enter should use submitOrEnqueue
	result, cmd := m.submitOrEnqueue()

	if cmd != nil {
		t.Errorf("cmd = %v; want nil", cmd)
	}
	if len(result.promptQueue) != 1 || result.promptQueue[0] != "shift queued" {
		t.Errorf("promptQueue = %v; want [shift queued]", result.promptQueue)
	}
}

// --- Alt+Enter tests ---

func TestAppModel_AltEnter_EnqueuesWhenIdle(t *testing.T) {
	m := NewAppModel(testDeps())
	m.agentRunning = false
	m.editor = m.editor.SetText("force queued")

	result, cmd := m.enqueuePrompt()

	if cmd != nil {
		t.Errorf("cmd = %v; want nil (alt+enter always enqueues, never submits)", cmd)
	}
	if len(result.promptQueue) != 1 || result.promptQueue[0] != "force queued" {
		t.Errorf("promptQueue = %v; want [force queued]", result.promptQueue)
	}
	if result.editor.Text() != "" {
		t.Errorf("editor should be cleared; got %q", result.editor.Text())
	}
	if result.footer.queuedCount != 1 {
		t.Errorf("footer.queuedCount = %d; want 1", result.footer.queuedCount)
	}
}

func TestAppModel_AltEnter_EnqueuesWhenAgentRunning(t *testing.T) {
	m := NewAppModel(testDeps())
	m.agentRunning = true
	m.editor = m.editor.SetText("alt queued running")

	result, cmd := m.enqueuePrompt()

	if cmd != nil {
		t.Errorf("cmd = %v; want nil", cmd)
	}
	if len(result.promptQueue) != 1 || result.promptQueue[0] != "alt queued running" {
		t.Errorf("promptQueue = %v; want [alt queued running]", result.promptQueue)
	}
	if result.footer.queuedCount != 1 {
		t.Errorf("footer.queuedCount = %d; want 1", result.footer.queuedCount)
	}
}

func TestAppModel_AltEnter_EmptyEditorNoOp(t *testing.T) {
	m := NewAppModel(testDeps())
	// editor is empty

	result, cmd := m.enqueuePrompt()

	if cmd != nil {
		t.Errorf("cmd = %v; want nil", cmd)
	}
	if len(result.promptQueue) != 0 {
		t.Errorf("promptQueue should be empty; got %v", result.promptQueue)
	}
}

func TestAppModel_AltEnter_MultipleEnqueues(t *testing.T) {
	m := NewAppModel(testDeps())

	prompts := []string{"first", "second", "third"}
	for _, p := range prompts {
		m.editor = m.editor.SetText(p)
		result, _ := m.enqueuePrompt()
		m = result
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
