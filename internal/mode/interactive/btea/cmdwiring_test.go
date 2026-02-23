// ABOUTME: Tests for command wiring infrastructure (buildCommandContext + side effects)
// ABOUTME: Verifies that cmdSideEffects signals are correctly produced by command callbacks

package btea

import (
	"testing"

	"github.com/mauromedda/pi-coding-agent-go/internal/commands"
	"github.com/mauromedda/pi-coding-agent-go/internal/permission"
	"github.com/mauromedda/pi-coding-agent-go/pkg/ai"
)

func TestBuildCommandContext_ReturnsNonNil(t *testing.T) {
	t.Parallel()

	m := newTestAppModel()
	ctx, effects := m.buildCommandContext()

	if ctx == nil {
		t.Fatal("expected non-nil CommandContext")
	}
	if effects == nil {
		t.Fatal("expected non-nil cmdSideEffects")
	}
}

func TestBuildCommandContext_PopulatesBasicFields(t *testing.T) {
	t.Parallel()

	m := newTestAppModel()
	ctx, _ := m.buildCommandContext()

	if ctx.Version == "" {
		t.Error("expected non-empty Version")
	}
	if ctx.Mode == "" {
		t.Error("expected non-empty Mode")
	}
}

func TestBuildCommandContext_ExitSetsQuit(t *testing.T) {
	t.Parallel()

	m := newTestAppModel()
	ctx, effects := m.buildCommandContext()

	if ctx.ExitFn == nil {
		t.Fatal("ExitFn should not be nil")
	}
	ctx.ExitFn()
	if !effects.quit {
		t.Error("expected quit=true after ExitFn")
	}
}

func TestBuildCommandContext_ClearSetsClearTUI(t *testing.T) {
	t.Parallel()

	m := newTestAppModel()
	ctx, effects := m.buildCommandContext()

	if ctx.ClearHistory == nil {
		t.Fatal("ClearHistory should not be nil")
	}
	ctx.ClearHistory()
	if !effects.clearTUI {
		t.Error("expected clearTUI=true after ClearHistory")
	}

	if ctx.ClearTUI == nil {
		t.Fatal("ClearTUI should not be nil")
	}
	ctx.ClearTUI()
	if !effects.clearTUI {
		t.Error("expected clearTUI=true after ClearTUI")
	}
}

func TestBuildCommandContext_ToggleModeSetsFlag(t *testing.T) {
	t.Parallel()

	m := newTestAppModel()
	ctx, effects := m.buildCommandContext()

	if ctx.ToggleMode == nil {
		t.Fatal("ToggleMode should not be nil")
	}
	ctx.ToggleMode()
	if !effects.modeToggled {
		t.Error("expected modeToggled=true after ToggleMode")
	}
}

func TestBuildCommandContext_GetModeReturnsCurrentMode(t *testing.T) {
	t.Parallel()

	m := newTestAppModel()
	m.mode = ModeEdit
	ctx, _ := m.buildCommandContext()

	if ctx.GetMode == nil {
		t.Fatal("GetMode should not be nil")
	}
	// After ToggleMode, mode should flip
	ctx.ToggleMode()
	got := ctx.GetMode()
	if got != "Plan" {
		t.Errorf("expected mode 'Plan' after toggle from Edit, got %q", got)
	}
}

func TestApplyEffects_Quit(t *testing.T) {
	t.Parallel()

	m := newTestAppModel()
	effects := &cmdSideEffects{quit: true}

	_, cmd := m.applyEffects(effects, "goodbye")
	if cmd == nil {
		t.Fatal("expected non-nil tea.Cmd for quit")
	}
}

func TestApplyEffects_ClearTUI(t *testing.T) {
	t.Parallel()

	m := newTestAppModel()
	// Add some content
	am := NewAssistantMsgModel()
	m.content = append(m.content, am)
	m.messages = append(m.messages, testUserMessage())
	m.totalInputTokens = 100
	m.totalOutputTokens = 200

	effects := &cmdSideEffects{clearTUI: true}
	result, cmd := m.applyEffects(effects, "cleared")

	resultModel := result.(AppModel)
	if len(resultModel.content) != 0 {
		t.Errorf("expected content cleared, got %d items", len(resultModel.content))
	}
	if len(resultModel.messages) != 0 {
		t.Errorf("expected messages cleared, got %d", len(resultModel.messages))
	}
	if resultModel.totalInputTokens != 0 {
		t.Error("expected totalInputTokens reset to 0")
	}
	if resultModel.totalOutputTokens != 0 {
		t.Error("expected totalOutputTokens reset to 0")
	}
	if cmd != nil {
		t.Error("expected nil cmd for clear (not quit)")
	}
}

func TestApplyEffects_ResultShowsAssistantMsg(t *testing.T) {
	t.Parallel()

	m := newTestAppModel()
	effects := &cmdSideEffects{}

	result, _ := m.applyEffects(effects, "some output")
	resultModel := result.(AppModel)

	if len(resultModel.content) == 0 {
		t.Fatal("expected content to have assistant message")
	}
	last, ok := resultModel.content[len(resultModel.content)-1].(*AssistantMsgModel)
	if !ok {
		t.Fatal("expected last content to be *AssistantMsgModel")
	}
	if last.text.String() == "" {
		t.Error("expected assistant message to contain result text")
	}
}

func TestLastAssistantText_Empty(t *testing.T) {
	t.Parallel()

	m := newTestAppModel()
	got := m.lastAssistantText()
	if got != "" {
		t.Errorf("expected empty string, got %q", got)
	}
}

func TestLastAssistantText_FindsLastAssistant(t *testing.T) {
	t.Parallel()

	m := newTestAppModel()
	am := NewAssistantMsgModel()
	am.Update(AgentTextMsg{Text: "hello world"})
	m.content = append(m.content, am)

	got := m.lastAssistantText()
	if got != "hello world" {
		t.Errorf("expected 'hello world', got %q", got)
	}
}

// --- Test helpers ---

func testUserMessage() ai.Message {
	return ai.NewTextMessage(ai.RoleUser, "test message")
}

func newTestAppModel() AppModel {
	return AppModel{
		sh:          &shared{},
		mode:        ModeEdit,
		editor:      NewEditorModel(),
		footer:      NewFooterModel(),
		cmdRegistry: commands.NewRegistry(),
		deps: AppDeps{
			Version:        "test-v0.1",
			PermissionMode: permission.ModeNormal,
		},
	}
}
