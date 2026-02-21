// ABOUTME: Tests for interactive mode App: construction, mode toggle, key routing
// ABOUTME: Uses VirtualTerminal for isolated TUI testing without real terminal

package interactive

import (
	"strings"
	"testing"

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

	// Line 2: token stats (left) + model (right)
	line2 := buf.Lines[1]
	if !strings.Contains(line2, "gpt-4o") {
		t.Errorf("footer line2 should contain model, got %q", line2)
	}
	if !strings.Contains(line2, "\u219112k") {
		t.Errorf("footer line2 should contain input tokens, got %q", line2)
	}
	if !strings.Contains(line2, "\u21938k") {
		t.Errorf("footer line2 should contain output tokens, got %q", line2)
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
