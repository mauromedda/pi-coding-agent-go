// ABOUTME: Tests for interactive mode App: construction, mode toggle, key routing
// ABOUTME: Uses VirtualTerminal for isolated TUI testing without real terminal

package interactive

import (
	"fmt"
	"strings"
	"testing"

	"github.com/mauromedda/pi-coding-agent-go/internal/config"
	"github.com/mauromedda/pi-coding-agent-go/internal/mode/interactive/components"
	"github.com/mauromedda/pi-coding-agent-go/internal/permission"
	"github.com/mauromedda/pi-coding-agent-go/pkg/ai"
	tui "github.com/mauromedda/pi-coding-agent-go/pkg/tui"
	"github.com/mauromedda/pi-coding-agent-go/pkg/tui/component"
	"github.com/mauromedda/pi-coding-agent-go/pkg/tui/key"
	"github.com/mauromedda/pi-coding-agent-go/pkg/tui/terminal"
)

func TestNewFromDeps(t *testing.T) {
	t.Parallel()

	vt := terminal.NewVirtualTerminal(80, 24)
	checker := permission.NewChecker(permission.ModeNormal, nil)

	app := NewFromDeps(AppDeps{
		Terminal: vt,
		Model:    &ai.Model{Name: "test-model", Api: ai.ApiOpenAI},
		Tools:    nil,
		Checker:  checker,
		Version:  "test",
	})

	if app.Mode() != ModePlan {
		t.Errorf("expected initial mode ModePlan, got %v", app.Mode())
	}
	if app.model.Name != "test-model" {
		t.Errorf("expected model name 'test-model', got %q", app.model.Name)
	}
}

func TestToggleMode(t *testing.T) {
	t.Parallel()

	vt := terminal.NewVirtualTerminal(80, 24)
	checker := permission.NewChecker(permission.ModeNormal, nil)
	app := New(vt, 80, 24, checker)

	if app.Mode() != ModePlan {
		t.Fatal("expected initial ModePlan")
	}

	app.ToggleMode()
	if app.Mode() != ModeEdit {
		t.Errorf("expected ModeEdit after toggle, got %v", app.Mode())
	}
	if checker.Mode() != permission.ModeNormal {
		t.Errorf("expected permission ModeNormal, got %v", checker.Mode())
	}

	app.ToggleMode()
	if app.Mode() != ModePlan {
		t.Errorf("expected ModePlan after second toggle, got %v", app.Mode())
	}
	if checker.Mode() != permission.ModePlan {
		t.Errorf("expected permission ModePlan, got %v", checker.Mode())
	}
}

func TestSetYoloMode(t *testing.T) {
	t.Parallel()

	vt := terminal.NewVirtualTerminal(80, 24)
	checker := permission.NewChecker(permission.ModeNormal, nil)
	app := New(vt, 80, 24, checker)

	app.SetYoloMode()

	if app.Mode() != ModeEdit {
		t.Errorf("expected ModeEdit in yolo, got %v", app.Mode())
	}
	if checker.Mode() != permission.ModeYolo {
		t.Errorf("expected ModeYolo, got %v", checker.Mode())
	}
}

func TestSetAcceptEditsMode(t *testing.T) {
	t.Parallel()

	vt := terminal.NewVirtualTerminal(80, 24)
	checker := permission.NewChecker(permission.ModeNormal, nil)
	app := New(vt, 80, 24, checker)

	app.SetAcceptEditsMode()

	if app.Mode() != ModeEdit {
		t.Errorf("expected ModeEdit, got %v", app.Mode())
	}
	if checker.Mode() != permission.ModeAcceptEdits {
		t.Errorf("expected ModeAcceptEdits, got %v", checker.Mode())
	}
}

func TestToggleMode_PreservesPermissionMode(t *testing.T) {
	t.Parallel()

	vt := terminal.NewVirtualTerminal(80, 24)
	checker := permission.NewChecker(permission.ModeAcceptEdits, nil)
	app := New(vt, 80, 24, checker)

	// Start in AcceptEdits mode via SetAcceptEditsMode
	app.SetAcceptEditsMode()
	if checker.Mode() != permission.ModeAcceptEdits {
		t.Fatal("precondition: expected ModeAcceptEdits")
	}

	// Toggle to Plan
	app.ToggleMode()
	if app.Mode() != ModePlan {
		t.Errorf("expected ModePlan, got %v", app.Mode())
	}
	if checker.Mode() != permission.ModePlan {
		t.Errorf("expected permission ModePlan, got %v", checker.Mode())
	}

	// Toggle back to Edit: should restore ModeAcceptEdits
	app.ToggleMode()
	if app.Mode() != ModeEdit {
		t.Errorf("expected ModeEdit, got %v", app.Mode())
	}
	if checker.Mode() != permission.ModeAcceptEdits {
		t.Errorf("expected ModeAcceptEdits restored, got %v", checker.Mode())
	}
}

func TestModeLabel_SubMode(t *testing.T) {
	t.Parallel()

	vt := terminal.NewVirtualTerminal(80, 24)
	checker := permission.NewChecker(permission.ModeAcceptEdits, nil)
	app := New(vt, 80, 24, checker)

	app.SetAcceptEditsMode()

	label := app.ModeLabel()
	if !strings.Contains(label, "EDIT") {
		t.Errorf("expected label to contain EDIT, got %q", label)
	}
	if !strings.Contains(label, "accept-edits") {
		t.Errorf("expected label to contain sub-mode, got %q", label)
	}
}

func TestModeLabel(t *testing.T) {
	t.Parallel()

	vt := terminal.NewVirtualTerminal(80, 24)
	checker := permission.NewChecker(permission.ModeNormal, nil)
	app := New(vt, 80, 24, checker)

	if app.ModeLabel() == "" {
		t.Error("expected non-empty mode label")
	}
}

func TestStatusLine(t *testing.T) {
	t.Parallel()

	vt := terminal.NewVirtualTerminal(80, 24)
	checker := permission.NewChecker(permission.ModeNormal, nil)
	app := New(vt, 80, 24, checker)

	line := app.StatusLine("test-model", 0.0042)
	if line == "" {
		t.Error("expected non-empty status line")
	}
}

func TestOnKey_CtrlD_Exits(t *testing.T) {
	t.Parallel()

	vt := terminal.NewVirtualTerminal(80, 24)
	checker := permission.NewChecker(permission.ModeNormal, nil)
	app := NewFromDeps(AppDeps{
		Terminal: vt,
		Model:    &ai.Model{Name: "test"},
		Checker:  checker,
	})
	app.editor = nil // no editor wired; we just test the exit path
	app.footer = nil

	// Ctrl+D should cancel context
	app.onKey(key.Key{Type: key.KeyCtrlD})

	select {
	case <-app.ctx.Done():
		// expected
	default:
		t.Error("expected context to be cancelled after Ctrl+D")
	}
}

func TestOnKey_BackTab_TogglesMode(t *testing.T) {
	t.Parallel()

	vt := terminal.NewVirtualTerminal(80, 24)
	checker := permission.NewChecker(permission.ModeNormal, nil)
	app := NewFromDeps(AppDeps{
		Terminal: vt,
		Model:    &ai.Model{Name: "test"},
		Checker:  checker,
	})
	// Set up minimal editor/footer so onKey doesn't panic
	app.tui.Start()
	defer app.tui.Stop()

	if app.Mode() != ModePlan {
		t.Fatal("expected initial ModePlan")
	}

	app.onKey(key.Key{Type: key.KeyBackTab})

	if app.Mode() != ModeEdit {
		t.Errorf("expected ModeEdit after BackTab, got %v", app.Mode())
	}
}

func TestFormatTokenCount(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name  string
		input int
		want  string
	}{
		{"small", 500, "500"},
		{"thousand", 1500, "2k"},
		{"large_k", 12345, "12k"},
		{"million", 1_200_000, "1.2M"},
		{"zero", 0, "0"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			if got := formatTokenCount(tt.input); got != tt.want {
				t.Errorf("formatTokenCount(%d) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}

func TestUpdateFooter_WithTokenStats(t *testing.T) {
	t.Parallel()

	vt := terminal.NewVirtualTerminal(80, 24)
	checker := permission.NewChecker(permission.ModeNormal, nil)
	app := NewFromDeps(AppDeps{
		Terminal: vt,
		Model:    &ai.Model{Name: "gpt-4o"},
		Checker:  checker,
	})
	app.footer = components.NewFooter()
	app.totalInputTokens.Store(12000)
	app.totalOutputTokens.Store(8000)
	app.gitBranch = "main"

	app.updateFooter()

	// Render footer to buffer and check both lines
	buf := tui.AcquireBuffer()
	defer tui.ReleaseBuffer(buf)
	app.footer.Render(buf, 80)

	if buf.Len() != 2 {
		t.Fatalf("expected 2 footer lines, got %d", buf.Len())
	}

	// Line 1: cwd + (branch)
	line1 := buf.Lines[0]
	if !strings.Contains(line1, "main") {
		t.Errorf("footer line1 should contain git branch, got %q", line1)
	}

	// Line 1 now also contains model name
	if !strings.Contains(line1, "gpt-4o") {
		t.Errorf("footer line1 should contain model, got %q", line1)
	}

	// Line 2: token stats (left) + permission mode
	line2 := buf.Lines[1]
	if !strings.Contains(line2, "\u219112k") {
		t.Errorf("footer line2 should contain input tokens, got %q", line2)
	}
	if !strings.Contains(line2, "\u21938k") {
		t.Errorf("footer line2 should contain output tokens, got %q", line2)
	}
}

func TestUpdateFooter_IncludesModeLabel(t *testing.T) {
	t.Parallel()

	vt := terminal.NewVirtualTerminal(80, 24)
	checker := permission.NewChecker(permission.ModeNormal, nil)
	app := NewFromDeps(AppDeps{
		Terminal: vt,
		Model:    &ai.Model{Name: "test-model"},
		Checker:  checker,
	})
	app.footer = components.NewFooter()

	// Initial mode is Plan
	app.updateFooter()

	buf := tui.AcquireBuffer()
	defer tui.ReleaseBuffer(buf)
	app.footer.Render(buf, 80)

	if buf.Len() < 2 {
		t.Fatalf("expected 2 footer lines, got %d", buf.Len())
	}

	line2 := buf.Lines[1]
	if !strings.Contains(line2, "PLAN") {
		t.Errorf("footer line2 should contain mode label PLAN, got %q", line2)
	}
}

func TestHandleFileMentionInput_Tab_AcceptsSelection(t *testing.T) {
	t.Parallel()

	vt := terminal.NewVirtualTerminal(80, 24)
	checker := permission.NewChecker(permission.ModeNormal, nil)
	app := NewFromDeps(AppDeps{
		Terminal: vt,
		Model:    &ai.Model{Name: "test"},
		Checker:  checker,
	})
	app.editor = component.NewEditor()
	app.footer = components.NewFooter()
	app.tui.Start()
	defer app.tui.Stop()

	// Populate file mention selector with test items
	app.fileMentionSelector.SetItems([]component.FileInfo{
		{Path: "/tmp/main.go", RelPath: "main.go"},
		{Path: "/tmp/util.go", RelPath: "util.go"},
	})
	app.fileMentionVisible = true

	// Tab should accept the selected file
	handled := app.handleFileMentionInput(key.Key{Type: key.KeyTab})
	if !handled {
		t.Fatal("expected Tab to be handled when file mention is visible")
	}

	// File mention should be hidden after Tab
	if app.fileMentionVisible {
		t.Error("expected fileMentionVisible to be false after Tab")
	}

	// Editor should contain the selected file path
	text := app.editor.Text()
	if !strings.Contains(text, "main.go") {
		t.Errorf("expected editor to contain 'main.go', got %q", text)
	}
}

func TestHandleFileMentionInput_UpDown_NavigatesList(t *testing.T) {
	t.Parallel()

	vt := terminal.NewVirtualTerminal(80, 24)
	checker := permission.NewChecker(permission.ModeNormal, nil)
	app := NewFromDeps(AppDeps{
		Terminal: vt,
		Model:    &ai.Model{Name: "test"},
		Checker:  checker,
	})
	app.editor = component.NewEditor()
	app.footer = components.NewFooter()
	app.editorSep = components.NewSeparator()
	app.editorSepBot = components.NewSeparator()
	app.tui.Start()
	defer app.tui.Stop()

	app.fileMentionSelector.SetItems([]component.FileInfo{
		{Path: "/tmp/a.go", RelPath: "a.go"},
		{Path: "/tmp/b.go", RelPath: "b.go"},
		{Path: "/tmp/c.go", RelPath: "c.go"},
	})
	app.fileMentionVisible = true

	// Initially selected: a.go (index 0)
	if got := app.fileMentionSelector.SelectedRelPath(); got != "a.go" {
		t.Fatalf("expected initial selection 'a.go', got %q", got)
	}

	// Down should move to b.go
	handled := app.handleFileMentionInput(key.Key{Type: key.KeyDown})
	if !handled {
		t.Fatal("expected Down to be handled")
	}
	if got := app.fileMentionSelector.SelectedRelPath(); got != "b.go" {
		t.Errorf("expected 'b.go' after Down, got %q", got)
	}

	// Down again should move to c.go
	app.handleFileMentionInput(key.Key{Type: key.KeyDown})
	if got := app.fileMentionSelector.SelectedRelPath(); got != "c.go" {
		t.Errorf("expected 'c.go' after second Down, got %q", got)
	}

	// Up should move back to b.go
	app.handleFileMentionInput(key.Key{Type: key.KeyUp})
	if got := app.fileMentionSelector.SelectedRelPath(); got != "b.go" {
		t.Errorf("expected 'b.go' after Up, got %q", got)
	}
}

func TestHandleFileMentionInput_Backspace_TrimsFilter(t *testing.T) {
	t.Parallel()

	vt := terminal.NewVirtualTerminal(80, 24)
	checker := permission.NewChecker(permission.ModeNormal, nil)
	app := NewFromDeps(AppDeps{
		Terminal: vt,
		Model:    &ai.Model{Name: "test"},
		Checker:  checker,
	})
	app.tui.Start()
	defer app.tui.Stop()

	app.fileMentionSelector.SetItems([]component.FileInfo{
		{Path: "/tmp/main.go", RelPath: "main.go"},
	})
	app.fileMentionVisible = true

	// Set a filter
	app.fileMentionSelector.SetFilter("mai")

	// Backspace should trim the filter, not pass through to editor
	handled := app.handleFileMentionInput(key.Key{Type: key.KeyBackspace})
	if !handled {
		t.Fatal("expected Backspace to be handled when file mention is visible")
	}

	got := app.fileMentionSelector.Filter()
	if got != "ma" {
		t.Errorf("expected filter 'ma' after backspace, got %q", got)
	}
}

func TestNewAssistantSegment_InsertsBeforeEditor(t *testing.T) {
	t.Parallel()

	vt := terminal.NewVirtualTerminal(80, 24)
	checker := permission.NewChecker(permission.ModeNormal, nil)
	app := NewFromDeps(AppDeps{
		Terminal: vt,
		Model:    &ai.Model{Name: "test"},
		Checker:  checker,
	})
	app.editor = component.NewEditor()
	app.footer = components.NewFooter()
	app.editorSep = components.NewSeparator()
	app.editorSepBot = components.NewSeparator()
	app.tui.Start()
	defer app.tui.Stop()

	// Set up container with editor/footer at the bottom
	container := app.tui.Container()
	container.Add(app.editorSep)
	container.Add(app.editor)
	container.Add(app.editorSepBot)
	container.Add(app.footer)

	// Create a lazy assistant segment
	msg := app.newAssistantSegment()
	if msg == nil {
		t.Fatal("expected non-nil AssistantMessage")
	}

	// Verify ordering: msg should be before editorSep
	children := container.Children()
	if len(children) != 5 {
		t.Fatalf("expected 5 children, got %d", len(children))
	}
	if children[0] != msg {
		t.Error("expected AssistantMessage at index 0")
	}
	if children[1] != app.editorSep {
		t.Error("expected editorSep at index 1")
	}
	if children[2] != app.editor {
		t.Error("expected editor at index 2")
	}
	if children[3] != app.editorSepBot {
		t.Error("expected editorSepBot at index 3")
	}
	if children[4] != app.footer {
		t.Error("expected footer at index 4")
	}
}

func TestNewAssistantSegment_AddsToolCallsInline(t *testing.T) {
	t.Parallel()

	vt := terminal.NewVirtualTerminal(80, 24)
	checker := permission.NewChecker(permission.ModeNormal, nil)
	app := NewFromDeps(AppDeps{
		Terminal: vt,
		Model:    &ai.Model{Name: "test"},
		Checker:  checker,
	})
	app.editor = component.NewEditor()
	app.footer = components.NewFooter()
	app.editorSep = components.NewSeparator()
	app.editorSepBot = components.NewSeparator()
	app.tui.Start()
	defer app.tui.Stop()

	// Simulate: text → tool → text interleaving
	msg1 := app.newAssistantSegment()
	msg1.AppendText("Let me check...")

	// Simulate tool call: tool calls are now added inline to assistant messages
	tc := components.NewToolCall("read", `{"path":"foo.go"}`)
	msg1.AddToolCall(tc)

	// Second text segment after tool
	msg2 := app.newAssistantSegment()
	msg2.AppendText("Found it.")

	// Tool calls are now inline in AssistantMessage, not as separate container children
	// So we expect: msg1 (with inline tool call), msg2, editorSep, editor, editorSepBot, footer
	container := app.tui.Container()
	children := container.Children()
	if len(children) != 6 {
		t.Fatalf("expected 6 children (msg1, msg2, editorSep, editor, editorSepBot, footer), got %d", len(children))
	}
	if children[0] != msg1 {
		t.Error("expected msg1 at index 0")
	}
	if children[1] != msg2 {
		t.Error("expected msg2 at index 1")
	}
	if children[2] != app.editorSep {
		t.Error("expected editorSep at index 2")
	}

	// Verify the tool call is inline in msg1
	tcList := msg1.GetToolCalls()
	if len(tcList) != 1 {
		t.Fatalf("expected 1 tool call in msg1, got %d", len(tcList))
	}
	if tcList[0] != tc {
		t.Error("expected tool call to be in msg1")
	}
}

func TestSubmitPrompt_DoesNotPreCreateAssistant(t *testing.T) {
	t.Parallel()

	vt := terminal.NewVirtualTerminal(80, 24)
	checker := permission.NewChecker(permission.ModeNormal, nil)
	app := NewFromDeps(AppDeps{
		Terminal: vt,
		Model:    &ai.Model{Name: "test"},
		Checker:  checker,
	})
	app.editor = component.NewEditor()
	app.footer = components.NewFooter()
	app.editorSep = components.NewSeparator()
	app.editorSepBot = components.NewSeparator()
	app.tui.Start()
	defer app.tui.Stop()

	container := app.tui.Container()
	container.Add(app.editorSep)
	container.Add(app.editor)
	container.Add(app.editorSepBot)
	container.Add(app.footer)

	app.submitPrompt("hello")

	// Wait for the agent goroutine to finish (no provider → returns immediately)
	for app.agentRunning.Load() {
		// spin briefly
	}

	// a.current should be nil: no pre-created assistant message
	if app.current != nil {
		t.Error("expected a.current to be nil after submitPrompt (lazy creation)")
	}

	// Container should have: UserMessage, editorSep, editor, editorSepBot, footer
	// (no AssistantMessage since provider is nil and lazy creation skips it)
	children := container.Children()
	foundAssistant := false
	for _, c := range children {
		if _, ok := c.(*components.AssistantMessage); ok {
			foundAssistant = true
		}
	}
	if foundAssistant {
		t.Error("expected no AssistantMessage in container (lazy creation, no provider)")
	}
}

func TestUpdateFooter_ShowsContextPct(t *testing.T) {
	t.Parallel()

	vt := terminal.NewVirtualTerminal(80, 24)
	checker := permission.NewChecker(permission.ModeNormal, nil)
	app := NewFromDeps(AppDeps{
		Terminal: vt,
		Model:    &ai.Model{Name: "test-model", MaxTokens: 200000},
		Checker:  checker,
	})
	app.footer = components.NewFooter()

	// Simulate 100k input tokens out of 200k max = 50%
	app.lastContextTokens.Store(100000)
	app.updateFooter()

	buf := tui.AcquireBuffer()
	defer tui.ReleaseBuffer(buf)
	app.footer.Render(buf, 80)

	if buf.Len() < 2 {
		t.Fatalf("expected 2 footer lines, got %d", buf.Len())
	}
	line2 := buf.Lines[1]
	if !strings.Contains(line2, "ctx 50%") {
		t.Errorf("footer line2 should contain 'ctx 50%%', got %q", line2)
	}
}

func TestUpdateFooter_ContextPctZeroWhenNoTokens(t *testing.T) {
	t.Parallel()

	vt := terminal.NewVirtualTerminal(80, 24)
	checker := permission.NewChecker(permission.ModeNormal, nil)
	app := NewFromDeps(AppDeps{
		Terminal: vt,
		Model:    &ai.Model{Name: "test-model", MaxTokens: 200000},
		Checker:  checker,
	})
	app.footer = components.NewFooter()

	// No tokens stored (default 0)
	app.updateFooter()

	buf := tui.AcquireBuffer()
	defer tui.ReleaseBuffer(buf)
	app.footer.Render(buf, 80)

	if buf.Len() < 2 {
		t.Fatalf("expected 2 footer lines, got %d", buf.Len())
	}
	line2 := buf.Lines[1]
	if strings.Contains(line2, "ctx") {
		t.Errorf("footer should not show context pct when no tokens, got %q", line2)
	}
}

func TestAutoCompact_TriggersAt80Pct(t *testing.T) {
	t.Parallel()

	// Build 15 messages (more than keepRecentMessages=10 so Compact actually shrinks)
	msgs := make([]ai.Message, 15)
	for i := range msgs {
		msgs[i] = ai.NewTextMessage(ai.RoleUser, fmt.Sprintf("msg %d", i))
	}

	vt := terminal.NewVirtualTerminal(80, 24)
	checker := permission.NewChecker(permission.ModeNormal, nil)
	app := NewFromDeps(AppDeps{
		Terminal: vt,
		Model:    &ai.Model{Name: "test", MaxTokens: 200000},
		Checker:  checker,
	})
	app.footer = components.NewFooter()
	app.messages = msgs

	// 160k out of 200k = 80% => should trigger compaction
	app.lastContextTokens.Store(160000)

	app.autoCompactIfNeeded()

	if len(app.messages) >= 15 {
		t.Errorf("expected messages to be compacted from 15, got %d", len(app.messages))
	}
}

func TestAutoCompact_DoesNotTriggerBelow80Pct(t *testing.T) {
	t.Parallel()

	msgs := make([]ai.Message, 15)
	for i := range msgs {
		msgs[i] = ai.NewTextMessage(ai.RoleUser, fmt.Sprintf("msg %d", i))
	}

	vt := terminal.NewVirtualTerminal(80, 24)
	checker := permission.NewChecker(permission.ModeNormal, nil)
	app := NewFromDeps(AppDeps{
		Terminal: vt,
		Model:    &ai.Model{Name: "test", MaxTokens: 200000},
		Checker:  checker,
	})
	app.footer = components.NewFooter()
	app.messages = msgs

	// 100k out of 200k = 50% => should NOT trigger
	app.lastContextTokens.Store(100000)

	app.autoCompactIfNeeded()

	if len(app.messages) != 15 {
		t.Errorf("expected messages unchanged at 15, got %d", len(app.messages))
	}
}

func TestHandleSlashCommand_WiresCompactFn(t *testing.T) {
	t.Parallel()

	vt := terminal.NewVirtualTerminal(80, 24)
	checker := permission.NewChecker(permission.ModeNormal, nil)
	app := NewFromDeps(AppDeps{
		Terminal: vt,
		Model:    &ai.Model{Name: "test", MaxTokens: 200000},
		Checker:  checker,
	})
	app.editor = component.NewEditor()
	app.footer = components.NewFooter()
	app.editorSep = components.NewSeparator()
	app.editorSepBot = components.NewSeparator()
	app.tui.Start()
	defer app.tui.Stop()

	container := app.tui.Container()
	container.Add(app.editorSep)
	container.Add(app.editor)
	container.Add(app.editorSepBot)
	container.Add(app.footer)

	// Add enough messages so Compact() actually shrinks
	for i := 0; i < 15; i++ {
		app.messages = append(app.messages, ai.NewTextMessage(ai.RoleUser, fmt.Sprintf("msg %d", i)))
	}

	app.handleSlashCommand(container, "/compact")

	// After /compact, messages should be compacted
	if len(app.messages) >= 15 {
		t.Errorf("expected messages compacted from 15, got %d", len(app.messages))
	}
}

func TestAutoCompact_CustomThreshold(t *testing.T) {
	t.Parallel()

	msgs := make([]ai.Message, 15)
	for i := range msgs {
		msgs[i] = ai.NewTextMessage(ai.RoleUser, fmt.Sprintf("msg %d", i))
	}

	vt := terminal.NewVirtualTerminal(80, 24)
	checker := permission.NewChecker(permission.ModeNormal, nil)
	app := NewFromDeps(AppDeps{
		Terminal:             vt,
		Model:                &ai.Model{Name: "test", MaxTokens: 200000},
		Checker:              checker,
		AutoCompactThreshold: 50,
	})
	app.footer = components.NewFooter()
	app.messages = msgs

	// 100k out of 200k = 50% => should trigger at custom threshold 50
	app.lastContextTokens.Store(100000)

	app.autoCompactIfNeeded()

	if len(app.messages) >= 15 {
		t.Errorf("expected messages to be compacted from 15 at 50%% threshold, got %d", len(app.messages))
	}
}

func TestAutoCompact_CustomThreshold_DoesNotTriggerBelow(t *testing.T) {
	t.Parallel()

	msgs := make([]ai.Message, 15)
	for i := range msgs {
		msgs[i] = ai.NewTextMessage(ai.RoleUser, fmt.Sprintf("msg %d", i))
	}

	vt := terminal.NewVirtualTerminal(80, 24)
	checker := permission.NewChecker(permission.ModeNormal, nil)
	app := NewFromDeps(AppDeps{
		Terminal:             vt,
		Model:                &ai.Model{Name: "test", MaxTokens: 200000},
		Checker:              checker,
		AutoCompactThreshold: 50,
	})
	app.footer = components.NewFooter()
	app.messages = msgs

	// 80k out of 200k = 40% => should NOT trigger at 50% threshold
	app.lastContextTokens.Store(80000)

	app.autoCompactIfNeeded()

	if len(app.messages) != 15 {
		t.Errorf("expected messages unchanged at 15 (below 50%% threshold), got %d", len(app.messages))
	}
}

func TestCompactionThreshold_Default(t *testing.T) {
	t.Parallel()

	vt := terminal.NewVirtualTerminal(80, 24)
	checker := permission.NewChecker(permission.ModeNormal, nil)
	app := NewFromDeps(AppDeps{
		Terminal: vt,
		Model:    &ai.Model{Name: "test"},
		Checker:  checker,
	})

	if got := app.compactionThreshold(); got != 80 {
		t.Errorf("compactionThreshold() = %d, want 80 (default)", got)
	}
}

func TestCompactionThreshold_Custom(t *testing.T) {
	t.Parallel()

	vt := terminal.NewVirtualTerminal(80, 24)
	checker := permission.NewChecker(permission.ModeNormal, nil)
	app := NewFromDeps(AppDeps{
		Terminal:             vt,
		Model:                &ai.Model{Name: "test"},
		Checker:              checker,
		AutoCompactThreshold: 60,
	})

	if got := app.compactionThreshold(); got != 60 {
		t.Errorf("compactionThreshold() = %d, want 60", got)
	}
}

func TestCompactionThreshold_OutOfRange(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		threshold int
		want      int
	}{
		{"zero falls back to default", 0, 80},
		{"negative falls back to default", -5, 80},
		{"over 100 falls back to default", 150, 80},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			vt := terminal.NewVirtualTerminal(80, 24)
			checker := permission.NewChecker(permission.ModeNormal, nil)
			app := NewFromDeps(AppDeps{
				Terminal:             vt,
				Model:                &ai.Model{Name: "test"},
				Checker:              checker,
				AutoCompactThreshold: tt.threshold,
			})
			if got := app.compactionThreshold(); got != tt.want {
				t.Errorf("compactionThreshold() = %d, want %d", got, tt.want)
			}
		})
	}
}

func TestHandleFileMentionInput_Backspace_EmptyFilter_Hides(t *testing.T) {
	t.Parallel()

	vt := terminal.NewVirtualTerminal(80, 24)
	checker := permission.NewChecker(permission.ModeNormal, nil)
	app := NewFromDeps(AppDeps{
		Terminal: vt,
		Model:    &ai.Model{Name: "test"},
		Checker:  checker,
	})
	app.tui.Start()
	defer app.tui.Stop()

	app.fileMentionVisible = true

	// Backspace on empty filter should hide the selector
	handled := app.handleFileMentionInput(key.Key{Type: key.KeyBackspace})
	if !handled {
		t.Fatal("expected Backspace to be handled")
	}

	if app.fileMentionVisible {
		t.Error("expected file mention to be hidden after backspace on empty filter")
	}
}

func TestOnKey_AltT_CyclesThinkingLevel(t *testing.T) {
	t.Parallel()

	vt := terminal.NewVirtualTerminal(80, 24)
	checker := permission.NewChecker(permission.ModeNormal, nil)
	app := NewFromDeps(AppDeps{
		Terminal: vt,
		Model:    &ai.Model{Name: "test"},
		Checker:  checker,
	})
	app.editor = component.NewEditor()
	app.footer = components.NewFooter()
	app.editorSep = components.NewSeparator()
	app.editorSepBot = components.NewSeparator()
	app.tui.Start()
	defer app.tui.Stop()

	// Initial thinking level should be Off (0)
	if app.thinkingLevel != config.ThinkingOff {
		t.Fatalf("expected initial thinking level Off, got %v", app.thinkingLevel)
	}

	// Press alt+t 5 times: Off -> Minimal -> Low -> Medium -> High -> XHigh
	expected := []config.ThinkingLevel{
		config.ThinkingMinimal,
		config.ThinkingLow,
		config.ThinkingMedium,
		config.ThinkingHigh,
		config.ThinkingXHigh,
	}

	altT := key.Key{Type: key.KeyRune, Rune: 't', Alt: true}

	for i, want := range expected {
		app.onKey(altT)
		if app.thinkingLevel != want {
			t.Errorf("after press %d: expected thinking level %v, got %v", i+1, want, app.thinkingLevel)
		}
	}

	// Press alt+t once more: XHigh -> Off (wrap)
	app.onKey(altT)
	if app.thinkingLevel != config.ThinkingOff {
		t.Errorf("expected thinking level to wrap to Off, got %v", app.thinkingLevel)
	}
}

func TestOnKey_Slash_ShowsCommandPalette(t *testing.T) {
	t.Parallel()

	vt := terminal.NewVirtualTerminal(80, 24)
	checker := permission.NewChecker(permission.ModeNormal, nil)
	app := NewFromDeps(AppDeps{
		Terminal: vt,
		Model:    &ai.Model{Name: "test"},
		Checker:  checker,
	})
	app.editor = component.NewEditor()
	app.footer = components.NewFooter()
	app.editorSep = components.NewSeparator()
	app.editorSepBot = components.NewSeparator()
	app.tui.Start()
	defer app.tui.Stop()

	// Type '/' to trigger command palette
	app.onKey(key.Key{Type: key.KeyRune, Rune: '/'})

	if !app.cmdPaletteVisible {
		t.Error("expected command palette to be visible after typing '/'")
	}
}

func TestCommandPalette_FilterAndAccept(t *testing.T) {
	t.Parallel()

	vt := terminal.NewVirtualTerminal(80, 24)
	checker := permission.NewChecker(permission.ModeNormal, nil)
	app := NewFromDeps(AppDeps{
		Terminal: vt,
		Model:    &ai.Model{Name: "test"},
		Checker:  checker,
	})
	app.editor = component.NewEditor()
	app.footer = components.NewFooter()
	app.editorSep = components.NewSeparator()
	app.editorSepBot = components.NewSeparator()
	app.tui.Start()
	defer app.tui.Stop()

	// Show palette
	app.onKey(key.Key{Type: key.KeyRune, Rune: '/'})
	if !app.cmdPaletteVisible {
		t.Fatal("expected palette visible")
	}

	// Type 'h' 'e' 'l' to filter to "help"
	app.onKey(key.Key{Type: key.KeyRune, Rune: 'h'})
	app.onKey(key.Key{Type: key.KeyRune, Rune: 'e'})
	app.onKey(key.Key{Type: key.KeyRune, Rune: 'l'})

	// Accept with Enter — should put "/help" in editor
	app.onKey(key.Key{Type: key.KeyEnter})

	if app.cmdPaletteVisible {
		t.Error("expected palette hidden after Enter")
	}

	text := app.editor.Text()
	if text != "/help" {
		t.Errorf("expected editor text '/help', got %q", text)
	}
}

func TestCommandPalette_EscapeCancels(t *testing.T) {
	t.Parallel()

	vt := terminal.NewVirtualTerminal(80, 24)
	checker := permission.NewChecker(permission.ModeNormal, nil)
	app := NewFromDeps(AppDeps{
		Terminal: vt,
		Model:    &ai.Model{Name: "test"},
		Checker:  checker,
	})
	app.editor = component.NewEditor()
	app.footer = components.NewFooter()
	app.editorSep = components.NewSeparator()
	app.editorSepBot = components.NewSeparator()
	app.tui.Start()
	defer app.tui.Stop()

	// Show palette
	app.onKey(key.Key{Type: key.KeyRune, Rune: '/'})
	if !app.cmdPaletteVisible {
		t.Fatal("expected palette visible")
	}

	// Escape should close without inserting
	app.onKey(key.Key{Type: key.KeyEscape})

	if app.cmdPaletteVisible {
		t.Error("expected palette hidden after Escape")
	}

	text := app.editor.Text()
	if text != "" {
		t.Errorf("expected empty editor after Escape, got %q", text)
	}
}

func TestSubmitPrompt_BangEscape(t *testing.T) {
	t.Parallel()

	vt := terminal.NewVirtualTerminal(80, 24)
	checker := permission.NewChecker(permission.ModeNormal, nil)
	app := NewFromDeps(AppDeps{
		Terminal: vt,
		Model:    &ai.Model{Name: "test"},
		Checker:  checker,
	})
	app.editor = component.NewEditor()
	app.footer = components.NewFooter()
	app.editorSep = components.NewSeparator()
	app.editorSepBot = components.NewSeparator()
	app.tui.Start()
	defer app.tui.Stop()

	container := app.tui.Container()
	container.Add(app.editorSep)
	container.Add(app.editor)
	container.Add(app.editorSepBot)
	container.Add(app.footer)

	// Submit "!echo hello" — should be handled as bash escape
	app.submitPrompt("!echo hello")

	// Wait for any goroutine to finish
	for app.agentRunning.Load() {
	}

	// The command should have been dispatched as a shell command, not sent to agent.
	// Messages should include the user message but not trigger agent
	// (since there's no provider, agent would error anyway).
	// Check that the user message "!echo hello" was displayed
	children := container.Children()
	foundUserMsg := false
	for _, c := range children {
		if _, ok := c.(*components.UserMessage); ok {
			foundUserMsg = true
		}
	}
	if !foundUserMsg {
		t.Error("expected UserMessage in container for bang command")
	}
}

func TestCommandPalette_BackspaceTrimsFilter(t *testing.T) {
	t.Parallel()

	vt := terminal.NewVirtualTerminal(80, 24)
	checker := permission.NewChecker(permission.ModeNormal, nil)
	app := NewFromDeps(AppDeps{
		Terminal: vt,
		Model:    &ai.Model{Name: "test"},
		Checker:  checker,
	})
	app.editor = component.NewEditor()
	app.footer = components.NewFooter()
	app.editorSep = components.NewSeparator()
	app.editorSepBot = components.NewSeparator()
	app.tui.Start()
	defer app.tui.Stop()

	// Show palette and type "he"
	app.onKey(key.Key{Type: key.KeyRune, Rune: '/'})
	app.onKey(key.Key{Type: key.KeyRune, Rune: 'h'})
	app.onKey(key.Key{Type: key.KeyRune, Rune: 'e'})

	// Backspace should trim to "h"
	app.onKey(key.Key{Type: key.KeyBackspace})

	if !app.cmdPaletteVisible {
		t.Error("expected palette still visible after backspace")
	}

	// Backspace again to empty filter
	app.onKey(key.Key{Type: key.KeyBackspace})

	// Backspace on empty filter should close palette
	app.onKey(key.Key{Type: key.KeyBackspace})

	if app.cmdPaletteVisible {
		t.Error("expected palette hidden after backspace on empty filter")
	}
}

func TestCopyLastAssistantMessage_NoMessages(t *testing.T) {
	t.Parallel()

	vt := terminal.NewVirtualTerminal(80, 24)
	checker := permission.NewChecker(permission.ModeNormal, nil)
	app := NewFromDeps(AppDeps{
		Terminal: vt,
		Model:    &ai.Model{Name: "test"},
		Checker:  checker,
	})

	result, err := app.copyLastAssistantMessage()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(result, "No assistant messages") {
		t.Errorf("expected 'No assistant messages', got %q", result)
	}
}

func TestCopyLastAssistantMessage_FindsLastAssistant(t *testing.T) {
	t.Parallel()

	vt := terminal.NewVirtualTerminal(80, 24)
	checker := permission.NewChecker(permission.ModeNormal, nil)
	app := NewFromDeps(AppDeps{
		Terminal: vt,
		Model:    &ai.Model{Name: "test"},
		Checker:  checker,
	})

	// Add mixed messages
	app.messages = []ai.Message{
		ai.NewTextMessage(ai.RoleUser, "hello"),
		ai.NewTextMessage(ai.RoleAssistant, "first reply"),
		ai.NewTextMessage(ai.RoleUser, "more"),
		ai.NewTextMessage(ai.RoleAssistant, "second reply"),
	}

	// Should find "second reply" (the last assistant message)
	// We can't test clipboard write without the actual binary, but we can
	// verify it doesn't error on macOS where pbcopy exists
	result, err := app.copyLastAssistantMessage()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(result, "Copied") {
		t.Errorf("expected 'Copied' result, got %q", result)
	}
}

func TestOnKey_ShiftCtrlP_CyclesModel(t *testing.T) {
	t.Parallel()

	vt := terminal.NewVirtualTerminal(80, 24)
	checker := permission.NewChecker(permission.ModeNormal, nil)
	scopedModels := &config.ScopedModelsConfig{
		Models: []config.ScopedModel{
			{Name: "model-a"},
			{Name: "model-b"},
			{Name: "model-c"},
		},
		Default: "model-a",
	}
	app := NewFromDeps(AppDeps{
		Terminal:     vt,
		Model:        &ai.Model{Name: "model-a"},
		Checker:      checker,
		ScopedModels: scopedModels,
	})
	app.editor = component.NewEditor()
	app.footer = components.NewFooter()
	app.editorSep = components.NewSeparator()
	app.editorSepBot = components.NewSeparator()
	app.tui.Start()
	defer app.tui.Stop()

	shiftCtrlP := key.Key{Type: key.KeyRune, Rune: 'p', Ctrl: true, Shift: true}

	// Cycle: model-a -> model-b
	app.onKey(shiftCtrlP)
	if app.model.Name != "model-b" {
		t.Errorf("expected model-b after first cycle, got %q", app.model.Name)
	}

	// Cycle: model-b -> model-c
	app.onKey(shiftCtrlP)
	if app.model.Name != "model-c" {
		t.Errorf("expected model-c after second cycle, got %q", app.model.Name)
	}

	// Cycle: model-c -> model-a (wrap)
	app.onKey(shiftCtrlP)
	if app.model.Name != "model-a" {
		t.Errorf("expected model-a after wrap, got %q", app.model.Name)
	}
}

func TestOnKey_ShiftCtrlP_NoopWithoutScopedModels(t *testing.T) {
	t.Parallel()

	vt := terminal.NewVirtualTerminal(80, 24)
	checker := permission.NewChecker(permission.ModeNormal, nil)
	app := NewFromDeps(AppDeps{
		Terminal: vt,
		Model:    &ai.Model{Name: "test-model"},
		Checker:  checker,
		// No ScopedModels
	})
	app.editor = component.NewEditor()
	app.footer = components.NewFooter()
	app.editorSep = components.NewSeparator()
	app.editorSepBot = components.NewSeparator()
	app.tui.Start()
	defer app.tui.Stop()

	shiftCtrlP := key.Key{Type: key.KeyRune, Rune: 'p', Ctrl: true, Shift: true}

	// Should not panic; model unchanged
	app.onKey(shiftCtrlP)
	if app.model.Name != "test-model" {
		t.Errorf("expected model unchanged, got %q", app.model.Name)
	}
}

func TestOnKey_AltT_UpdatesFooter(t *testing.T) {
	t.Parallel()

	vt := terminal.NewVirtualTerminal(80, 24)
	checker := permission.NewChecker(permission.ModeNormal, nil)
	app := NewFromDeps(AppDeps{
		Terminal: vt,
		Model:    &ai.Model{Name: "test"},
		Checker:  checker,
	})
	app.editor = component.NewEditor()
	app.footer = components.NewFooter()
	app.editorSep = components.NewSeparator()
	app.editorSepBot = components.NewSeparator()
	app.tui.Start()
	defer app.tui.Stop()

	altT := key.Key{Type: key.KeyRune, Rune: 't', Alt: true}

	// Press alt+t to cycle to Minimal
	app.onKey(altT)

	// Render footer and check it shows the thinking level
	buf := tui.AcquireBuffer()
	defer tui.ReleaseBuffer(buf)
	app.footer.Render(buf, 80)

	if buf.Len() < 2 {
		t.Fatalf("expected 2 footer lines, got %d", buf.Len())
	}
	line2 := buf.Lines[1]
	if !strings.Contains(line2, "minimal") {
		t.Errorf("footer line2 should contain thinking level 'minimal', got %q", line2)
	}
}
