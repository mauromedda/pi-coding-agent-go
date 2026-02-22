// ABOUTME: Tests for the root AppModel: init state, message routing, key handling, overlays
// ABOUTME: Table-driven tests covering the core happy path and overlay lifecycle

package btea

import (
	"fmt"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/mauromedda/pi-coding-agent-go/internal/config"
	"github.com/mauromedda/pi-coding-agent-go/pkg/ai"
)

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
	// Execute the command; it should return a gitBranchMsg
	msg := cmd()
	if _, ok := msg.(gitBranchMsg); !ok {
		t.Errorf("Init cmd returned %T; want gitBranchMsg", msg)
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
	if _, ok := model.content[1].(AssistantMsgModel); !ok {
		t.Errorf("content[1] = %T; want AssistantMsgModel", model.content[1])
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
	m.overlay = NewCmdPaletteModel(nil)

	result, _ := m.Update(CmdPaletteSelectMsg{Name: "help"})
	model := result.(AppModel)

	if model.overlay != nil {
		t.Error("overlay should be nil after CmdPaletteSelectMsg")
	}
	if model.editor.Text() != "/help" {
		t.Errorf("editor text = %q; want %q", model.editor.Text(), "/help")
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

func TestAppModel_EnterWhileRunningDoesNotSubmit(t *testing.T) {
	m := NewAppModel(testDeps())
	m.agentRunning = true
	m.editor = m.editor.SetText("new prompt")

	key := tea.KeyMsg{Type: tea.KeyEnter}
	result, _ := m.Update(key)
	model := result.(AppModel)

	// Should NOT add a new UserMsgModel
	for _, c := range model.content {
		if _, ok := c.(UserMsgModel); ok {
			t.Error("should not submit prompt while agent is running")
		}
	}
}
