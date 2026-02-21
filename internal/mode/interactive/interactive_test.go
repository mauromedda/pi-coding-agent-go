// ABOUTME: Tests for interactive mode App: construction, mode toggle, key routing
// ABOUTME: Uses VirtualTerminal for isolated TUI testing without real terminal

package interactive

import (
	"testing"

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
