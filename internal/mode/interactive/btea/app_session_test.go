// ABOUTME: Tests for session persistence wiring in the Bubble Tea TUI
// ABOUTME: Verifies user/assistant messages are persisted and session commands are wired

package btea

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/mauromedda/pi-coding-agent-go/internal/session"
	"github.com/mauromedda/pi-coding-agent-go/pkg/ai"
)

// testSession creates a session backed by a temp directory for isolated testing.
func testSession(t *testing.T) *session.Session {
	t.Helper()

	dir := t.TempDir()
	sessionsDir := filepath.Join(dir, "sessions")
	if err := os.MkdirAll(sessionsDir, 0o755); err != nil {
		t.Fatal(err)
	}

	// Create session file manually in temp dir to avoid using config.SessionsDir()
	path := filepath.Join(sessionsDir, "test-session.jsonl")
	f, err := os.OpenFile(path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o600)
	if err != nil {
		t.Fatal(err)
	}

	writer := session.NewWriterFromFile(f)

	return &session.Session{
		ID:       "test-session",
		Model:    &ai.Model{Name: "test-model", ID: "test-model", MaxOutputTokens: 4096},
		Messages: nil,
		Writer:   writer,
		CWD:      dir,
	}
}

func testDepsWithSession(t *testing.T) AppDeps {
	t.Helper()
	deps := testDeps()
	deps.Session = testSession(t)
	return deps
}

func TestAppModel_SubmitPromptPersistsUserMessage(t *testing.T) {
	deps := testDepsWithSession(t)
	m := NewAppModel(deps)
	m.editor = m.editor.SetText("tell me a joke")

	// Submit the prompt
	m, _ = m.submitPrompt("tell me a joke")

	// Session should have the user message
	if len(deps.Session.Messages) != 1 {
		t.Fatalf("session messages = %d; want 1", len(deps.Session.Messages))
	}
	if deps.Session.Messages[0].Role != ai.RoleUser {
		t.Errorf("message role = %q; want %q", deps.Session.Messages[0].Role, ai.RoleUser)
	}
}

func TestAppModel_SubmitPromptWithoutSession(t *testing.T) {
	// Session is nil; submitPrompt must not panic
	m := NewAppModel(testDeps())
	m.editor = m.editor.SetText("no session here")

	m, _ = m.submitPrompt("no session here")

	// Should still add to in-memory messages
	if len(m.messages) != 1 {
		t.Fatalf("messages = %d; want 1", len(m.messages))
	}
}

func TestAppModel_AgentDonePersistsAssistantMessages(t *testing.T) {
	deps := testDepsWithSession(t)
	m := NewAppModel(deps)
	m.agentRunning = true

	// Simulate agent returning messages
	msgs := []ai.Message{
		ai.NewTextMessage(ai.RoleUser, "hello"),
		ai.NewTextMessage(ai.RoleAssistant, "hi there"),
	}
	result, _ := m.Update(AgentDoneMsg{Messages: msgs})
	model := result.(AppModel)

	if model.agentRunning {
		t.Error("agentRunning = true; want false")
	}

	// Session should have persisted the assistant message
	if len(deps.Session.Messages) < 1 {
		t.Fatalf("session messages = %d; want at least 1 (assistant)", len(deps.Session.Messages))
	}

	// Find the assistant message in session
	hasAssistant := false
	for _, msg := range deps.Session.Messages {
		if msg.Role == ai.RoleAssistant {
			hasAssistant = true
			break
		}
	}
	if !hasAssistant {
		t.Error("session has no assistant message after AgentDoneMsg")
	}
}

func TestAppModel_SessionLoadedMsg(t *testing.T) {
	deps := testDepsWithSession(t)
	m := NewAppModel(deps)

	// Simulate loading session history
	history := []ai.Message{
		ai.NewTextMessage(ai.RoleUser, "previous question"),
		ai.NewTextMessage(ai.RoleAssistant, "previous answer"),
	}

	result, _ := m.Update(SessionLoadedMsg{
		SessionID: "test-session",
		Messages:  history,
	})
	model := result.(AppModel)

	// Messages should be loaded into conversation
	if len(model.messages) != 2 {
		t.Fatalf("messages = %d; want 2", len(model.messages))
	}

	// Content should include the restored messages as display models
	hasUser := false
	hasAssistant := false
	for _, c := range model.content {
		if _, ok := c.(UserMsgModel); ok {
			hasUser = true
		}
		if _, ok := c.(*AssistantMsgModel); ok {
			hasAssistant = true
		}
	}
	if !hasUser {
		t.Error("content missing restored user message")
	}
	if !hasAssistant {
		t.Error("content missing restored assistant message")
	}
}

func TestAppModel_SessionCommandsWired(t *testing.T) {
	deps := testDepsWithSession(t)
	m := NewAppModel(deps)

	ctx, _ := m.buildCommandContext()

	t.Run("RenameSession is wired", func(t *testing.T) {
		if ctx.RenameSession == nil {
			t.Error("RenameSession callback is nil; want non-nil")
		}
	})

	t.Run("ListSessionsFn is wired", func(t *testing.T) {
		if ctx.ListSessionsFn == nil {
			t.Error("ListSessionsFn callback is nil; want non-nil")
		}
	})
}

func TestAppModel_SessionSavedMsg(t *testing.T) {
	deps := testDepsWithSession(t)
	m := NewAppModel(deps)

	// SessionSavedMsg should be handled without panic
	result, cmd := m.Update(SessionSavedMsg{SessionID: "test-session"})
	if cmd != nil {
		t.Errorf("cmd = %v; want nil", cmd)
	}
	_ = result.(AppModel) // must not panic
}
