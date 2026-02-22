// ABOUTME: Tests for WelcomeMessage component rendering: banner, version, model, cwd, shortcuts, tools
// ABOUTME: Validates all rendered output fields via string containment on joined buffer lines

package components

import (
	"strings"
	"testing"

	"github.com/mauromedda/pi-coding-agent-go/pkg/tui"
	"github.com/mauromedda/pi-coding-agent-go/pkg/tui/width"
)

func renderWelcome(version, model, cwd string, toolCount int) string {
	w := NewWelcomeMessage(version, model, cwd, toolCount)
	buf := tui.AcquireBuffer()
	defer tui.ReleaseBuffer(buf)

	w.Render(buf, 80)

	return strings.Join(buf.Lines, "\n")
}

// renderWelcomeStripped returns the output with all ANSI codes removed.
func renderWelcomeStripped(version, model, cwd string, toolCount int) string {
	return width.StripANSI(renderWelcome(version, model, cwd, toolCount))
}

func TestWelcomeMessage_banner_present(t *testing.T) {
	t.Parallel()

	output := renderWelcome("1.0.0", "claude-opus-4-20250514", "/tmp", 5)
	if !strings.Contains(output, "π") {
		t.Error("rendered output should contain the pi character π")
	}
}

func TestWelcomeMessage_version_shown(t *testing.T) {
	t.Parallel()

	stripped := renderWelcomeStripped("1.0.0", "claude-opus-4-20250514", "/tmp", 5)
	if !strings.Contains(stripped, "pi-go") || !strings.Contains(stripped, "v1.0.0") {
		t.Errorf("rendered output should contain 'pi-go' and 'v1.0.0', got:\n%s", stripped)
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

	stripped := renderWelcomeStripped("1.0.0", "claude-opus-4-20250514", "/tmp", 5)

	// Shortcut keys and their descriptions (compact two-column format)
	shortcuts := []string{
		"escape",
		"interrupt",
		"ctrl+c",
		"clear",
		"ctrl+c twice",
		"exit",
		"ctrl+d",
		"shift+tab",
		"cycle mode",
		"/",
		"commands",
		"!",
		"run bash",
	}

	for _, s := range shortcuts {
		if !strings.Contains(stripped, s) {
			t.Errorf("rendered output should contain shortcut phrase %q, got:\n%s", s, stripped)
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

func TestWelcomeMessage_ascii_logo_multiline(t *testing.T) {
	t.Parallel()

	output := renderWelcome("1.0.0", "claude-opus-4-20250514", "/tmp", 5)
	// The welcome banner should have a multi-line ASCII π logo (at least 2 lines)
	lines := strings.Split(output, "\n")
	piLines := 0
	for _, l := range lines {
		stripped := strings.TrimSpace(width.StripANSI(l))
		// Logo lines contain box-drawing or π-related characters
		if strings.ContainsAny(stripped, "π╔╗╚╝║━┃┏┓┗┛─│╭╮╰╯▄▀█") {
			piLines++
		}
	}
	if piLines < 2 {
		t.Errorf("expected multi-line logo (at least 2 lines with logo chars), got %d", piLines)
	}
}

func TestWelcomeMessage_two_column_shortcuts(t *testing.T) {
	t.Parallel()

	output := renderWelcome("1.0.0", "claude-opus-4-20250514", "/tmp", 5)
	// Shortcuts should be aligned in columns (padded key + description)
	lines := strings.Split(output, "\n")
	paddedCount := 0
	for _, l := range lines {
		stripped := width.StripANSI(l)
		if strings.Contains(stripped, "escape") && strings.Contains(stripped, "interrupt") {
			paddedCount++
		}
		if strings.Contains(stripped, "ctrl+c") && strings.Contains(stripped, "clear") {
			paddedCount++
		}
	}
	if paddedCount < 2 {
		t.Errorf("expected at least 2 padded shortcut lines, got %d", paddedCount)
	}
}
