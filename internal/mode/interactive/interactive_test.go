// ABOUTME: Tests for interactive mode App: construction, mode toggle, key routing
// ABOUTME: Uses VirtualTerminal for isolated TUI testing without real terminal

package interactive

import (
	"strings"
	"testing"

	"github.com/mauromedda/pi-coding-agent-go/internal/mode/interactive/components"
	"github.com/mauromedda/pi-coding-agent-go/internal/permission"
	"github.com/mauromedda/pi-coding-agent-go/pkg/ai"
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

	// Render the footer to check content
	content := app.footer.Content()
	if !strings.Contains(content, "main") {
		t.Errorf("footer should contain git branch, got %q", content)
	}
	if !strings.Contains(content, "PLAN") {
		t.Errorf("footer should contain mode, got %q", content)
	}
	if !strings.Contains(content, "gpt-4o") {
		t.Errorf("footer should contain model, got %q", content)
	}
	if !strings.Contains(content, "\u219112k") {
		t.Errorf("footer should contain input tokens, got %q", content)
	}
	if !strings.Contains(content, "\u21938k") {
		t.Errorf("footer should contain output tokens, got %q", content)
	}
}
