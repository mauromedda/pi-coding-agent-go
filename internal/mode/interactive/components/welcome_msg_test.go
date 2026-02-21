// ABOUTME: Tests for WelcomeMessage component rendering: banner, version, model, cwd, shortcuts, tools
// ABOUTME: Validates all rendered output fields via string containment on joined buffer lines

package components

import (
	"strings"
	"testing"

	"github.com/mauromedda/pi-coding-agent-go/pkg/tui"
)

func renderWelcome(version, model, cwd string, toolCount int) string {
	w := NewWelcomeMessage(version, model, cwd, toolCount)
	buf := tui.AcquireBuffer()
	defer tui.ReleaseBuffer(buf)

	w.Render(buf, 80)

	return strings.Join(buf.Lines, "\n")
}

func TestWelcomeMessage_banner_present(t *testing.T) {
	t.Parallel()

	output := renderWelcome("1.0.0", "claude-opus-4-20250514", "/tmp", 5)
	if !strings.Contains(output, "\u03c0") {
		t.Error("rendered output should contain the pi character \u03c0")
	}
}

func TestWelcomeMessage_version_shown(t *testing.T) {
	t.Parallel()

	output := renderWelcome("1.0.0", "claude-opus-4-20250514", "/tmp", 5)
	if !strings.Contains(output, "pi-go v1.0.0") {
		t.Errorf("rendered output should contain 'pi-go v1.0.0', got:\n%s", output)
	}
}

func TestWelcomeMessage_model_shown(t *testing.T) {
	t.Parallel()

	output := renderWelcome("1.0.0", "claude-opus-4-20250514", "/tmp", 5)
	if !strings.Contains(output, "claude-opus-4-20250514") {
		t.Errorf("rendered output should contain model name, got:\n%s", output)
	}
}

func TestWelcomeMessage_cwd_shown(t *testing.T) {
	t.Parallel()

	output := renderWelcome("1.0.0", "claude-opus-4-20250514", "/home/user/project", 5)
	if !strings.Contains(output, "/home/user/project") {
		t.Errorf("rendered output should contain cwd, got:\n%s", output)
	}
}

func TestWelcomeMessage_shortcuts_present(t *testing.T) {
	t.Parallel()

	output := renderWelcome("1.0.0", "claude-opus-4-20250514", "/tmp", 5)

	shortcuts := []string{
		"escape",
		"to interrupt",
		"ctrl+c",
		"to clear",
		"ctrl+c twice",
		"to exit",
		"ctrl+d",
		"shift+tab",
		"to cycle mode",
		"/",
		"for commands",
		"!",
		"to run bash",
	}

	for _, s := range shortcuts {
		if !strings.Contains(output, s) {
			t.Errorf("rendered output should contain shortcut phrase %q, got:\n%s", s, output)
		}
	}
}

func TestWelcomeMessage_tool_count_shown(t *testing.T) {
	t.Parallel()

	output := renderWelcome("1.0.0", "claude-opus-4-20250514", "/tmp", 9)
	if !strings.Contains(output, "[Tools: 9 registered]") {
		t.Errorf("rendered output should contain '[Tools: 9 registered]', got:\n%s", output)
	}
}

func TestWelcomeMessage_dev_version_fallback(t *testing.T) {
	t.Parallel()

	output := renderWelcome("", "claude-opus-4-20250514", "/tmp", 5)
	// Should not panic and should contain a sensible version indicator
	if !strings.Contains(output, "pi-go") {
		t.Errorf("rendered output with empty version should still contain 'pi-go', got:\n%s", output)
	}
	if !strings.Contains(output, "dev") {
		t.Errorf("rendered output with empty version should contain 'dev' fallback, got:\n%s", output)
	}
}

func TestWelcomeMessage_zero_tools(t *testing.T) {
	t.Parallel()

	output := renderWelcome("1.0.0", "claude-opus-4-20250514", "/tmp", 0)
	if !strings.Contains(output, "[Tools: 0 registered]") {
		t.Errorf("rendered output should contain '[Tools: 0 registered]', got:\n%s", output)
	}
}
