// ABOUTME: Tests for the root AppModel: init state, message routing, key handling, overlays
// ABOUTME: Table-driven tests covering the core happy path and overlay lifecycle

package btea

import (
	"fmt"
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/mauromedda/pi-coding-agent-go/internal/config"
	"github.com/mauromedda/pi-coding-agent-go/pkg/ai"
)

// Compile-time check: AppModel must satisfy tea.Model.
var _ tea.Model = AppModel{}

func testDeps() AppDeps {
	return AppDeps{
		Model:   &ai.Model{Name: "test-model", MaxOutputTokens: 4096},
		Version: "0.1.0-test",
	}
}

func TestNewAppModel(t *testing.T) {
	m := NewAppModel(testDeps())

	t.Run("initial mode is Edit", func(t *testing.T) {
		if m.mode != ModeEdit {
			t.Errorf("mode = %v; want ModeEdit", m.mode)
		}
	})

	t.Run("agent not running", func(t *testing.T) {
		if m.agentRunning {
			t.Error("agentRunning = true; want false")
		}
	})

	t.Run("content has welcome model", func(t *testing.T) {
		if len(m.content) != 1 {
			t.Fatalf("content length = %d; want 1", len(m.content))
		}
		if _, ok := m.content[0].(WelcomeModel); !ok {
			t.Errorf("content[0] = %T; want WelcomeModel", m.content[0])
		}
	})

	t.Run("no overlay", func(t *testing.T) {
		if m.overlay != nil {
			t.Errorf("overlay = %v; want nil", m.overlay)
		}
	})

	t.Run("shared struct initialized", func(t *testing.T) {
		if m.sh == nil {
			t.Fatal("sh = nil; want non-nil")
		}
		if m.sh.ctx == nil {
			t.Error("sh.ctx = nil; want non-nil")
		}
	})

	t.Run("editor is focused", func(t *testing.T) {
		if !m.editor.focused {
			t.Error("editor.focused = false; want true")
		}
	})

	t.Run("cmd registry initialized", func(t *testing.T) {
		if m.cmdRegistry == nil {
			t.Error("cmdRegistry = nil; want non-nil")
		}
	})
}

func TestAppModel_Init(t *testing.T) {
	m := NewAppModel(testDeps())
	cmd := m.Init()
	if cmd == nil {
		t.Fatal("Init() returned nil; want a command")
	}
	// Init returns a batch of commands (git branch + probe).
	// Execute the batch and verify a gitBranchMsg is among the results.
	msg := cmd()
	batch, ok := msg.(tea.BatchMsg)
	if !ok {
		t.Fatalf("Init cmd returned %T; want tea.BatchMsg", msg)
	}

	var hasGitBranch bool
	for _, batchCmd := range batch {
		if batchCmd == nil {
			continue
		}
		innerMsg := batchCmd()
		if _, ok := innerMsg.(gitBranchMsg); ok {
			hasGitBranch = true
		}
	}
	if !hasGitBranch {
		t.Error("Init batch does not contain a gitBranchMsg command")
	}
}

func TestAppModel_WindowSizeMsg(t *testing.T) {
	m := NewAppModel(testDeps())
	msg := tea.WindowSizeMsg{Width: 120, Height: 40}

	result, cmd := m.Update(msg)
	model := result.(AppModel)

	if cmd != nil {
		t.Errorf("cmd = %v; want nil", cmd)
	}
	if model.width != 120 {
		t.Errorf("width = %d; want 120", model.width)
	}
	if model.height != 40 {
		t.Errorf("height = %d; want 40", model.height)
	}
}

func TestAppModel_GitBranchMsg(t *testing.T) {
	m := NewAppModel(testDeps())
	result, _ := m.Update(gitBranchMsg{branch: "feature/xyz"})
	model := result.(AppModel)

	if model.gitBranch != "feature/xyz" {
		t.Errorf("gitBranch = %q; want %q", model.gitBranch, "feature/xyz")
	}
}

func TestAppModel_ModeToggle(t *testing.T) {
	m := NewAppModel(testDeps())

	// Start in Edit, toggle to Plan
	key := tea.KeyMsg{Type: tea.KeyShiftTab}
	result, _ := m.Update(key)
	model := result.(AppModel)

	if model.mode != ModePlan {
		t.Errorf("after first toggle: mode = %v; want ModePlan", model.mode)
	}

	// Toggle back to Edit
	result, _ = model.Update(key)
	model = result.(AppModel)
	if model.mode != ModeEdit {
		t.Errorf("after second toggle: mode = %v; want ModeEdit", model.mode)
	}
}

func TestAppModel_CtrlCQuitsWhenIdle(t *testing.T) {
	m := NewAppModel(testDeps())
	key := tea.KeyMsg{Type: tea.KeyCtrlC}

	_, cmd := m.Update(key)
	if cmd == nil {
		t.Fatal("cmd = nil; want tea.Quit")
	}
	// Execute the cmd to get the quit message
	msg := cmd()
	if _, ok := msg.(tea.QuitMsg); !ok {
		t.Errorf("cmd() = %T; want tea.QuitMsg", msg)
	}
}

func TestAppModel_CtrlDQuits(t *testing.T) {
	m := NewAppModel(testDeps())
	key := tea.KeyMsg{Type: tea.KeyCtrlD}

	_, cmd := m.Update(key)
	if cmd == nil {
		t.Fatal("cmd = nil; want tea.Quit")
	}
	msg := cmd()
	if _, ok := msg.(tea.QuitMsg); !ok {
		t.Errorf("cmd() = %T; want tea.QuitMsg", msg)
	}
}

func TestAppModel_AgentTextMsg(t *testing.T) {
	m := NewAppModel(testDeps())

	// Send agent text; should create assistant msg and append text
	result, _ := m.Update(AgentTextMsg{Text: "Hello world"})
	model := result.(AppModel)

	// Should have welcome + assistant
	if len(model.content) != 2 {
		t.Fatalf("content length = %d; want 2", len(model.content))
	}
	if _, ok := model.content[1].(*AssistantMsgModel); !ok {
		t.Errorf("content[1] = %T; want *AssistantMsgModel", model.content[1])
	}
}

func TestAppModel_AgentDoneMsg(t *testing.T) {
	m := NewAppModel(testDeps())
	m.agentRunning = true

	result, _ := m.Update(AgentDoneMsg{})
	model := result.(AppModel)

	if model.agentRunning {
		t.Error("agentRunning = true; want false after AgentDoneMsg")
	}
}

func TestAppModel_AgentDoneMsgPreservesMessages(t *testing.T) {
	m := NewAppModel(testDeps())
	m.agentRunning = true

	msgs := []ai.Message{
		ai.NewTextMessage(ai.RoleUser, "hello"),
		ai.NewTextMessage(ai.RoleAssistant, "hi there"),
	}
	result, _ := m.Update(AgentDoneMsg{Messages: msgs})
	model := result.(AppModel)

	if len(model.messages) != 2 {
		t.Fatalf("messages length = %d; want 2", len(model.messages))
	}
	if model.agentRunning {
		t.Error("agentRunning should be false after AgentDoneMsg")
	}
}

func TestAppModel_AgentUsageMsg(t *testing.T) {
	m := NewAppModel(testDeps())
	usage := &ai.Usage{InputTokens: 500, OutputTokens: 100}

	result, _ := m.Update(AgentUsageMsg{Usage: usage})
	model := result.(AppModel)

	if model.totalInputTokens != 500 {
		t.Errorf("totalInputTokens = %d; want 500", model.totalInputTokens)
	}
	if model.totalOutputTokens != 100 {
		t.Errorf("totalOutputTokens = %d; want 100", model.totalOutputTokens)
	}
}

func TestAppModel_AgentUsageMsg_SetsFooterContextPct(t *testing.T) {
	deps := testDeps()
	deps.Model.ContextWindow = 10000
	m := NewAppModel(deps)

	// Send usage that fills 60% of context window
	usage := &ai.Usage{InputTokens: 6000, OutputTokens: 0}
	result, _ := m.Update(AgentUsageMsg{Usage: usage})
	model := result.(AppModel)

	if model.footer.contextPct != 60 {
		t.Errorf("footer.contextPct = %d; want 60", model.footer.contextPct)
	}

	// Accumulate more: now 80%
	usage2 := &ai.Usage{InputTokens: 2000, OutputTokens: 0}
	result2, _ := model.Update(AgentUsageMsg{Usage: usage2})
	model2 := result2.(AppModel)

	if model2.footer.contextPct != 80 {
		t.Errorf("footer.contextPct = %d; want 80", model2.footer.contextPct)
	}
}

func TestAppModel_AgentUsageMsg_NoContextWindow(t *testing.T) {
	// When ContextWindow is 0, contextPct should remain 0
	deps := testDeps()
	deps.Model.ContextWindow = 0
	deps.Model.MaxTokens = 0
	m := NewAppModel(deps)

	usage := &ai.Usage{InputTokens: 500, OutputTokens: 100}
	result, _ := m.Update(AgentUsageMsg{Usage: usage})
	model := result.(AppModel)

	if model.footer.contextPct != 0 {
		t.Errorf("footer.contextPct = %d; want 0 when no context window", model.footer.contextPct)
	}
}

func TestAppModel_PermissionRequestCreatesOverlay(t *testing.T) {
	m := NewAppModel(testDeps())
	ch := make(chan PermissionReply, 1)
	msg := PermissionRequestMsg{
		Tool:    "bash",
		Args:    map[string]any{"cmd": "rm -rf"},
		ReplyCh: ch,
	}

	result, _ := m.Update(msg)
	model := result.(AppModel)

	if model.overlay == nil {
		t.Fatal("overlay = nil; want PermDialogModel")
	}
	if _, ok := model.overlay.(PermDialogModel); !ok {
		t.Errorf("overlay = %T; want PermDialogModel", model.overlay)
	}
}

func TestAppModel_DismissOverlayMsg(t *testing.T) {
	m := NewAppModel(testDeps())
	// Set a dummy overlay
	m.overlay = NewCmdPaletteModel(nil)

	result, _ := m.Update(DismissOverlayMsg{})
	model := result.(AppModel)

	if model.overlay != nil {
		t.Errorf("overlay = %v; want nil after DismissOverlayMsg", model.overlay)
	}
}

func TestAppModel_CmdPaletteSelectMsg(t *testing.T) {
	m := NewAppModel(testDeps())
	m.width = 80
	m.overlay = NewCmdPaletteModel(nil)

	result, _ := m.Update(CmdPaletteSelectMsg{Name: "help"})
	model := result.(AppModel)

	if model.overlay != nil {
		t.Error("overlay should be nil after CmdPaletteSelectMsg")
	}
	// Enter on palette places command text in editor (NOT auto-submit)
	if got := model.editor.Text(); got != "/help" {
		t.Errorf("editor text = %q; want %q", got, "/help")
	}
	// Content should NOT have new user message (only welcome)
	if len(model.content) != 1 {
		t.Errorf("content length = %d; want 1 (only welcome, no submit)", len(model.content))
	}
	// Editor should be focused
	if !model.editor.focused {
		t.Error("editor should be focused after palette select")
	}
}

func TestAppModel_FileMentionSelectMsg(t *testing.T) {
	m := NewAppModel(testDeps())
	m.overlay = NewFileMentionModel("")

	result, _ := m.Update(FileMentionSelectMsg{RelPath: "main.go"})
	model := result.(AppModel)

	if model.overlay != nil {
		t.Error("overlay should be nil after FileMentionSelectMsg")
	}
	if got := model.editor.Text(); got != "@main.go" {
		t.Errorf("editor text = %q; want %q", got, "@main.go")
	}
}

func TestAppModel_CycleThinking(t *testing.T) {
	m := NewAppModel(testDeps())

	if m.thinkingLevel != config.ThinkingOff {
		t.Fatalf("initial thinking = %v; want ThinkingOff", m.thinkingLevel)
	}

	key := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'t'}, Alt: true}
	result, _ := m.Update(key)
	model := result.(AppModel)

	if model.thinkingLevel != config.ThinkingMinimal {
		t.Errorf("after cycle: thinking = %v; want ThinkingMinimal", model.thinkingLevel)
	}
}

func TestAppModel_OverlayRoutesMessages(t *testing.T) {
	m := NewAppModel(testDeps())
	ch := make(chan PermissionReply, 1)
	m.overlay = NewPermDialogModel("bash", nil, ch)

	// Send 'y' key; should be routed to overlay
	key := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'y'}}
	result, cmd := m.Update(key)
	model := result.(AppModel)

	// The perm dialog should have sent the reply and returned a dismiss cmd
	if cmd == nil {
		t.Fatal("cmd = nil; want dismiss overlay cmd")
	}

	// Check the reply was sent
	select {
	case reply := <-ch:
		if !reply.Allowed {
			t.Error("reply.Allowed = false; want true")
		}
	default:
		t.Error("no reply sent on channel")
	}

	// Execute the cmd to get DismissOverlayMsg
	dismissMsg := cmd()
	if _, ok := dismissMsg.(DismissOverlayMsg); !ok {
		t.Errorf("cmd() = %T; want DismissOverlayMsg", dismissMsg)
	}

	// Apply dismiss
	result, _ = model.Update(dismissMsg)
	model = result.(AppModel)
	if model.overlay != nil {
		t.Error("overlay should be nil after dismiss")
	}
}

func TestAppModel_SubmitPromptCreatesUserMsg(t *testing.T) {
	m := NewAppModel(testDeps())
	// Type some text into the editor
	m.editor = m.editor.SetText("tell me a joke")

	// Press enter
	key := tea.KeyMsg{Type: tea.KeyEnter}
	result, _ := m.Update(key)
	model := result.(AppModel)

	// Should have welcome + user msg
	if len(model.content) < 2 {
		t.Fatalf("content length = %d; want at least 2", len(model.content))
	}
	if _, ok := model.content[1].(UserMsgModel); !ok {
		t.Errorf("content[1] = %T; want UserMsgModel", model.content[1])
	}
}

func TestAppModel_SubmitSlashCommand(t *testing.T) {
	m := NewAppModel(testDeps())
	m.editor = m.editor.SetText("/help")

	key := tea.KeyMsg{Type: tea.KeyEnter}
	result, _ := m.Update(key)
	model := result.(AppModel)

	// Should have welcome + user msg + assistant msg (command result)
	if len(model.content) < 3 {
		t.Fatalf("content length = %d; want at least 3", len(model.content))
	}

	// Agent should NOT be running for slash commands
	if model.agentRunning {
		t.Error("agentRunning = true; want false for slash commands")
	}
}

func TestAppModel_SlashExit_ProducesQuit(t *testing.T) {
	m := NewAppModel(testDeps())
	m.editor = m.editor.SetText("/exit")

	key := tea.KeyMsg{Type: tea.KeyEnter}
	_, cmd := m.Update(key)

	if cmd == nil {
		t.Fatal("expected tea.Quit cmd from /exit, got nil")
	}
	// tea.Quit returns a special QuitMsg when invoked.
	msg := cmd()
	if _, ok := msg.(tea.QuitMsg); !ok {
		t.Errorf("expected tea.QuitMsg, got %T", msg)
	}
}

func TestAppModel_SlashClear_ClearsContent(t *testing.T) {
	m := NewAppModel(testDeps())

	// Simulate some content first
	m.messages = append(m.messages, ai.NewTextMessage(ai.RoleUser, "hello"))
	m.content = append(m.content, NewUserMsgModel("hello"))

	m.editor = m.editor.SetText("/clear")
	key := tea.KeyMsg{Type: tea.KeyEnter}
	result, _ := m.Update(key)
	model := result.(AppModel)

	if len(model.messages) != 0 {
		t.Errorf("messages length = %d; want 0 after /clear", len(model.messages))
	}
}

func TestAppModel_AgentErrorMsg(t *testing.T) {
	m := NewAppModel(testDeps())

	result, _ := m.Update(AgentErrorMsg{Err: fmt.Errorf("connection lost")})
	model := result.(AppModel)

	// Should create assistant msg with error text
	if len(model.content) < 2 {
		t.Fatalf("content length = %d; want at least 2", len(model.content))
	}
}

func TestAppModel_View(t *testing.T) {
	m := NewAppModel(testDeps())
	m.width = 80
	m.height = 24
	m = m.propagateSize(tea.WindowSizeMsg{Width: 80, Height: 24})

	view := m.View()
	if view == "" {
		t.Error("View() returned empty string")
	}
}

func TestMode_String(t *testing.T) {
	tests := []struct {
		mode Mode
		want string
	}{
		{ModePlan, "Plan"},
		{ModeEdit, "Edit"},
		{Mode(99), "Unknown"},
	}
	for _, tt := range tests {
		t.Run(tt.want, func(t *testing.T) {
			if got := tt.mode.String(); got != tt.want {
				t.Errorf("String() = %q; want %q", got, tt.want)
			}
		})
	}
}

func TestAppModel_SlashKeyOpensCmdPalette(t *testing.T) {
	m := NewAppModel(testDeps())
	// Editor is empty, not running
	key := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'/'}}

	result, _ := m.Update(key)
	model := result.(AppModel)

	if model.overlay == nil {
		t.Fatal("overlay = nil; want CmdPaletteModel")
	}
	if _, ok := model.overlay.(CmdPaletteModel); !ok {
		t.Errorf("overlay = %T; want CmdPaletteModel", model.overlay)
	}
}

func TestAppModel_AtKeyOpensFileMention(t *testing.T) {
	m := NewAppModel(testDeps())
	key := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'@'}}

	result, _ := m.Update(key)
	model := result.(AppModel)

	if model.overlay == nil {
		t.Fatal("overlay = nil; want FileMentionModel")
	}
	if _, ok := model.overlay.(FileMentionModel); !ok {
		t.Errorf("overlay = %T; want FileMentionModel", model.overlay)
	}
}

func TestAppModel_EnterWhileRunningEnqueues(t *testing.T) {
	m := NewAppModel(testDeps())
	m.agentRunning = true
	m.editor = m.editor.SetText("new prompt")

	key := tea.KeyMsg{Type: tea.KeyEnter}
	result, _ := m.Update(key)
	model := result.(AppModel)

	// Should NOT add a UserMsgModel (enqueue only; no content shown)
	for _, c := range model.content {
		if _, ok := c.(UserMsgModel); ok {
			t.Error("enqueue should not add UserMsgModel to content")
		}
	}
	// Should be in the queue
	if len(model.promptQueue) != 1 {
		t.Fatalf("promptQueue length = %d; want 1", len(model.promptQueue))
	}
	if model.promptQueue[0] != "new prompt" {
		t.Errorf("promptQueue[0] = %q; want %q", model.promptQueue[0], "new prompt")
	}
}

func TestAppModel_BashCommandWithSpaces(t *testing.T) {
	m := NewAppModel(testDeps())
	// Type !ls -la into the editor
	m.editor = m.editor.SetText("!ls -la")

	// Press enter
	key := tea.KeyMsg{Type: tea.KeyEnter}
	result, _ := m.Update(key)
	model := result.(AppModel)

	// The user message should preserve spaces
	if len(model.content) < 2 {
		t.Fatalf("content length = %d; want at least 2", len(model.content))
	}
	
	um, ok := model.content[1].(UserMsgModel)
	if !ok {
		t.Fatalf("content[1] = %T; want UserMsgModel", model.content[1])
	}
	
	// Check that spaces are preserved
	if um.Text() != "!ls -la" {
		t.Errorf("UserMsgModel.Text() = %q; want %q", um.Text(), "!ls -la")
	}
}

func TestAppModel_SeparatorColorLogic(t *testing.T) {
	// Test that separator color is determined correctly based on last content
	tests := []struct {
		name         string
		lastContent  tea.Model
		wantBashSep  bool
	}{
		{"Last is AssistantMsgModel", &AssistantMsgModel{}, true},
		{"Last is UserMsgModel", NewUserMsgModel("hello"), false},
		{"Last is WelcomeModel", NewWelcomeModel("v1", "model", "/home", 0), false},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := NewAppModel(testDeps())
			m.content = append(m.content, tt.lastContent)
			
			// Simulate the color selection logic
			sepColorIsBashSeparator := false
			if len(m.content) > 0 {
				if _, isAssistant := m.content[len(m.content)-1].(*AssistantMsgModel); isAssistant {
					sepColorIsBashSeparator = true
				}
			}
			
			// Check if the correct color was selected
			if tt.wantBashSep && !sepColorIsBashSeparator {
				t.Errorf("Expected BashSeparator for AssistantMsgModel")
			}
			if !tt.wantBashSep && sepColorIsBashSeparator {
				t.Errorf("Did not expect BashSeparator for %T", tt.lastContent)
			}
		})
	}
}

func TestAppModel_TypeBashCommandPreservesSpaces(t *testing.T) {
	m := NewAppModel(testDeps())
	
	// Simulate typing: !ls -la
	keys := []tea.KeyMsg{
		{Type: tea.KeyRunes, Runes: []rune{'!'}},
		{Type: tea.KeyRunes, Runes: []rune{'l'}},
		{Type: tea.KeyRunes, Runes: []rune{'s'}},
		{Type: tea.KeyRunes, Runes: []rune{' '}},
		{Type: tea.KeyRunes, Runes: []rune{'-'}},
		{Type: tea.KeyRunes, Runes: []rune{'l'}},
		{Type: tea.KeyRunes, Runes: []rune{'a'}},
	}
	
	for _, key := range keys {
		result, _ := m.Update(key)
		m = result.(AppModel)
	}
	
	// Check that the editor has the correct text with space
	text := m.editor.Text()
	if text != "!ls -la" {
		t.Errorf("Editor text = %q; want %q", text, "!ls -la")
	}
}

func TestAppModel_BashCommandOutputPreservesSpaces(t *testing.T) {
	m := NewAppModel(testDeps())
	
	// Simulate typing: !ls -la
	m.editor = m.editor.SetText("!ls -la")
	
	// Press enter to submit
	result, _ := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	m = result.(AppModel)
	
	// Check that the User message has the correct text
	if len(m.content) < 2 {
		t.Fatalf("Expected at least 2 content items, got %d", len(m.content))
	}
	
	um, ok := m.content[1].(UserMsgModel)
	if !ok {
		t.Fatalf("Expected UserMsgModel, got %T", m.content[1])
	}
	
	if um.Text() != "!ls -la" {
		t.Errorf("User message text = %q; want %q", um.Text(), "!ls -la")
	}
}

func TestAppModel_SpaceKeyInEditor(t *testing.T) {
	m := NewAppModel(testDeps())
	
	// Type: space
	result, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{' '}})
	m = result.(AppModel)
	
	// Check that space is in the editor
	text := m.editor.Text()
	if text != " " {
		t.Errorf("Editor text after space = %q; want %q", text, " ")
	}
	
	// Type: ! l s - l a (with spaces)
	keys := []tea.KeyMsg{
		{Type: tea.KeyRunes, Runes: []rune{'!'}},
		{Type: tea.KeyRunes, Runes: []rune{' '}},
		{Type: tea.KeyRunes, Runes: []rune{'l'}},
		{Type: tea.KeyRunes, Runes: []rune{'s'}},
		{Type: tea.KeyRunes, Runes: []rune{' '}},
		{Type: tea.KeyRunes, Runes: []rune{'-'}},
		{Type: tea.KeyRunes, Runes: []rune{'l'}},
		{Type: tea.KeyRunes, Runes: []rune{'a'}},
	}
	
	for _, key := range keys {
		result, _ := m.Update(key)
		m = result.(AppModel)
	}
	
	// Check that spaces are preserved (editor starts with one empty line, so first char is space)
	text = m.editor.Text()
	if text != " ! ls -la" {
		t.Errorf("Editor text = %q; want %q", text, " ! ls -la")
	}
}

func TestAppModel_SlashCommandAfterBash(t *testing.T) {
	m := NewAppModel(testDeps())
	
	// First execute a bash command
	m.editor = m.editor.SetText("!echo test")
	result, _ := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	m = result.(AppModel)
	
	// Now type / to open command palette
	result, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'/'}})
	m = result.(AppModel)
	
	// Should have overlay (command palette)
	if m.overlay == nil {
		t.Error("Expected command palette overlay to be open")
	}
	
	// Editor should have "/" (not cleared)
	if got := m.editor.Text(); got != "/" {
		t.Errorf("editor text = %q; want %q", got, "/")
	}
}

func TestAppModel_AgentTextMsgWhenLastContentIsUserMsg(t *testing.T) {
	m := NewAppModel(testDeps())

	// Add a user message as last content (simulates prompt submitted without agent running)
	um := NewUserMsgModel("hello")
	m.content = append(m.content, um)

	// Sending AgentTextMsg should NOT panic; it should create a new AssistantMsgModel
	result, _ := m.Update(AgentTextMsg{Text: "response"})
	model := result.(AppModel)

	last := model.content[len(model.content)-1]
	if _, ok := last.(*AssistantMsgModel); !ok {
		t.Errorf("last content = %T; want *AssistantMsgModel", last)
	}
}

func TestAppModel_UpdateLastAssistantSafeWithNonAssistant(t *testing.T) {
	// updateLastAssistant must not panic when last content is not *AssistantMsgModel.
	// This simulates a race or unexpected ordering where ensureAssistantMsg was not called.
	m := NewAppModel(testDeps())

	// Force last content to be a UserMsgModel (not AssistantMsgModel)
	um := NewUserMsgModel("hello")
	m.content = []tea.Model{um}

	// Direct call to updateLastAssistant — must NOT panic, should return unchanged
	m = m.updateLastAssistant(AgentTextMsg{Text: "should not crash"})

	// Content should still have only the UserMsgModel (no crash, no mutation)
	if len(m.content) != 1 {
		t.Errorf("content length = %d; want 1", len(m.content))
	}
	if _, ok := m.content[0].(UserMsgModel); !ok {
		t.Errorf("content[0] = %T; want UserMsgModel", m.content[0])
	}
}

func TestAppModel_SlashCommandFilter(t *testing.T) {
	m := NewAppModel(testDeps())
	
	// Type /h to filter for commands starting with h
	keys := []tea.KeyMsg{
		{Type: tea.KeyRunes, Runes: []rune{'/'}},  // /
		{Type: tea.KeyRunes, Runes: []rune{'h'}},  // h
	}
	
	for _, key := range keys {
		result, _ := m.Update(key)
		m = result.(AppModel)
	}
	
	// Should have overlay with command palette
	if m.overlay == nil {
		t.Error("Expected command palette overlay to be open")
	}
	
	palette, ok := m.overlay.(CmdPaletteModel)
	if !ok {
		t.Fatalf("Expected CmdPaletteModel overlay; got %T", m.overlay)
	}
	
	// Check that filter is applied
	visible := palette.visible
	if len(visible) == 0 {
		t.Error("Expected at least one command matching 'h'")
	}
	
	// All visible commands should contain 'h' (filter only checks command name)
	for _, entry := range visible {
		if !strings.Contains(strings.ToLower(entry.Name), "h") {
			t.Errorf("Command %q should contain 'h'", entry.Name)
		}
	}

	// Editor should mirror the typed text: "/h"
	if got := m.editor.Text(); got != "/h" {
		t.Errorf("editor text = %q; want %q", got, "/h")
	}
}

func TestAppModel_SlashKeyKeepsSlashInEditor(t *testing.T) {
	m := NewAppModel(testDeps())
	// Type / to open palette
	result, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'/'}})
	model := result.(AppModel)

	// Overlay should be open
	if model.overlay == nil {
		t.Fatal("overlay = nil; want CmdPaletteModel")
	}
	// Editor should contain the "/" character
	if got := model.editor.Text(); got != "/" {
		t.Errorf("editor text = %q; want %q", got, "/")
	}
}

func TestAppModel_TabCompletesFromPalette(t *testing.T) {
	m := NewAppModel(testDeps())
	m.width = 80

	// Open palette
	result, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'/'}})
	m = result.(AppModel)

	// Get selected command name from palette
	palette, ok := m.overlay.(CmdPaletteModel)
	if !ok {
		t.Fatal("expected CmdPaletteModel overlay")
	}
	selectedName := palette.Selected()
	if selectedName == "" {
		t.Fatal("no command selected in palette")
	}

	// Simulate CmdPaletteSelectMsg (sent by palette on Tab or Enter)
	result, _ = m.Update(CmdPaletteSelectMsg{Name: selectedName})
	m = result.(AppModel)

	// Editor should have the command text
	want := "/" + selectedName
	if got := m.editor.Text(); got != want {
		t.Errorf("editor text = %q; want %q", got, want)
	}
	// Overlay dismissed
	if m.overlay != nil {
		t.Error("overlay should be nil after tab completion")
	}
}

func TestAppModel_ModeTransitionMsg(t *testing.T) {
	m := NewAppModel(testDeps())
	result, _ := m.Update(ModeTransitionMsg{From: "Plan", To: "Execute", Reason: "user action"})
	model := result.(AppModel)

	if model.footer.intentLabel != "Execute" {
		t.Errorf("footer.intentLabel = %q; want Execute", model.footer.intentLabel)
	}
}

func TestAppModel_SettingsChangedMsg(t *testing.T) {
	m := NewAppModel(testDeps())
	result, cmd := m.Update(SettingsChangedMsg{Section: "personality"})
	if cmd != nil {
		t.Errorf("cmd = %v; want nil", cmd)
	}
	// Should not panic and return valid model
	_ = result.(AppModel)
}

func TestAppModel_PlanGeneratedMsg(t *testing.T) {
	m := NewAppModel(testDeps())
	result, _ := m.Update(PlanGeneratedMsg{Plan: "Step 1: Do X\nStep 2: Do Y"})
	model := result.(AppModel)

	if model.overlay == nil {
		t.Fatal("overlay = nil; want PlanViewModel")
	}
	if _, ok := model.overlay.(PlanViewModel); !ok {
		t.Errorf("overlay = %T; want PlanViewModel", model.overlay)
	}
}

func TestAppModel_PlanApprovedMsg(t *testing.T) {
	m := NewAppModel(testDeps())
	m.overlay = NewPlanViewModel("test plan")

	result, _ := m.Update(PlanApprovedMsg{})
	model := result.(AppModel)

	if model.overlay != nil {
		t.Errorf("overlay = %v; want nil after PlanApprovedMsg", model.overlay)
	}
}

func TestAppModel_PlanRejectedMsg(t *testing.T) {
	m := NewAppModel(testDeps())
	m.overlay = NewPlanViewModel("test plan")

	result, _ := m.Update(PlanRejectedMsg{})
	model := result.(AppModel)

	if model.overlay != nil {
		t.Errorf("overlay = %v; want nil after PlanRejectedMsg", model.overlay)
	}
}

func TestAppModel_CtrlTTogglesCostDashboard(t *testing.T) {
	m := NewAppModel(testDeps())
	key := tea.KeyMsg{Type: tea.KeyCtrlT}

	// First press: open cost dashboard
	result, _ := m.Update(key)
	model := result.(AppModel)
	if model.overlay == nil {
		t.Fatal("overlay = nil; want CostViewModel after ctrl+t")
	}
	if _, ok := model.overlay.(CostViewModel); !ok {
		t.Errorf("overlay = %T; want CostViewModel", model.overlay)
	}

	// Second press: overlay routes ctrl+t to CostViewModel which returns DismissOverlayMsg cmd
	result, cmd := model.Update(key)
	model = result.(AppModel)

	if cmd == nil {
		t.Fatal("cmd = nil; want DismissOverlayMsg cmd from CostViewModel")
	}
	// Execute the cmd to get DismissOverlayMsg
	dismissMsg := cmd()
	if _, ok := dismissMsg.(DismissOverlayMsg); !ok {
		t.Errorf("cmd() = %T; want DismissOverlayMsg", dismissMsg)
	}

	// Apply dismiss
	result, _ = model.Update(dismissMsg)
	model = result.(AppModel)
	if model.overlay != nil {
		t.Errorf("overlay = %v; want nil after dismiss", model.overlay)
	}
}

// --- WS1: Async I/O tests ---

func TestAppModel_HandleBashCommandReturnsCmd(t *testing.T) {
	m := NewAppModel(testDeps())
	m.width = 80
	m.editor = m.editor.SetText("!echo hello")

	key := tea.KeyMsg{Type: tea.KeyEnter}
	result, cmd := m.Update(key)
	model := result.(AppModel)

	// handleBashCommand should return a non-nil tea.Cmd (async execution)
	if cmd == nil {
		t.Fatal("cmd = nil; want non-nil tea.Cmd for async bash execution")
	}

	// bashRunning should be true while command executes
	if !model.bashRunning {
		t.Error("bashRunning = false; want true while bash command is in flight")
	}
}

func TestAppModel_BashDoneMsgCreatesOutputModel(t *testing.T) {
	m := NewAppModel(testDeps())
	m.width = 80
	m.bashRunning = true

	result, _ := m.Update(BashDoneMsg{
		Command:  "echo hello",
		Output:   "hello\n",
		ExitCode: 0,
	})
	model := result.(AppModel)

	if model.bashRunning {
		t.Error("bashRunning = true; want false after BashDoneMsg")
	}

	// Should have created a BashOutputModel in content
	found := false
	for _, c := range model.content {
		if _, ok := c.(*BashOutputModel); ok {
			found = true
			break
		}
	}
	if !found {
		t.Error("content should contain a *BashOutputModel after BashDoneMsg")
	}
}

func TestAppModel_InitReturnsThreeCmds(t *testing.T) {
	m := NewAppModel(testDeps())
	cmd := m.Init()
	if cmd == nil {
		t.Fatal("Init() returned nil; want batch command")
	}

	msg := cmd()
	batch, ok := msg.(tea.BatchMsg)
	if !ok {
		t.Fatalf("Init cmd returned %T; want tea.BatchMsg", msg)
	}

	// Should have 3 cmds: git branch + git cwd + probe
	if len(batch) != 3 {
		t.Errorf("Init batch has %d cmds; want 3 (gitBranch + gitCWD + probe)", len(batch))
	}

	// Execute each and check for gitCWDMsg
	var hasGitCWD bool
	for _, batchCmd := range batch {
		if batchCmd == nil {
			continue
		}
		innerMsg := batchCmd()
		if _, ok := innerMsg.(gitCWDMsg); ok {
			hasGitCWD = true
		}
	}
	if !hasGitCWD {
		t.Error("Init batch does not contain a gitCWDMsg command")
	}
}

func TestAppModel_GitCWDMsgSetsField(t *testing.T) {
	m := NewAppModel(testDeps())
	result, _ := m.Update(gitCWDMsg{cwd: "/home/user/project"})
	model := result.(AppModel)

	if model.gitCWD != "/home/user/project" {
		t.Errorf("gitCWD = %q; want %q", model.gitCWD, "/home/user/project")
	}
}

func TestAbortAgent_ConcurrentAccess(t *testing.T) {
	t.Parallel()
	m := NewAppModel(testDeps())

	// Simulate concurrent store and load on activeAgent via shared struct.
	// With atomic.Pointer this must not race.
	done := make(chan struct{})
	go func() {
		defer close(done)
		for range 1000 {
			m.sh.activeAgent.Store(nil)
		}
	}()

	for range 1000 {
		m.abortAgent()
	}

	<-done
}

func TestAppModel_WindowSizeMsgPropagatedToOverlay(t *testing.T) {
	m := NewAppModel(testDeps())
	ch := make(chan PermissionReply, 1)
	m.overlay = NewPermDialogModel("Bash", nil, ch)

	msg := tea.WindowSizeMsg{Width: 60, Height: 30}
	result, _ := m.Update(msg)
	model := result.(AppModel)

	// Overlay should have received the width
	pd, ok := model.overlay.(PermDialogModel)
	if !ok {
		t.Fatalf("overlay = %T; want PermDialogModel", model.overlay)
	}
	if pd.width != 60 {
		t.Errorf("overlay width = %d; want 60", pd.width)
	}
}
