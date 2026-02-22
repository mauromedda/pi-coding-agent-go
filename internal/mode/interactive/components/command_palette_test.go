// ABOUTME: Tests for CommandPalette overlay: filtering, rendering, selection
// ABOUTME: Table-driven tests verify command listing, fuzzy filter, and highlight

package components

import (
	"strings"
	"testing"

	"github.com/mauromedda/pi-coding-agent-go/pkg/tui"
	"github.com/mauromedda/pi-coding-agent-go/pkg/tui/width"
)

func TestCommandPalette_renders_commands(t *testing.T) {
	t.Parallel()

	cmds := []CommandEntry{
		{Name: "clear", Description: "Clear conversation"},
		{Name: "help", Description: "Show help"},
		{Name: "exit", Description: "Exit app"},
	}
	cp := NewCommandPalette(cmds)

	buf := tui.AcquireBuffer()
	defer tui.ReleaseBuffer(buf)
	cp.Render(buf, 60)

	output := strings.Join(buf.Lines, "\n")
	stripped := width.StripANSI(output)

	if !strings.Contains(stripped, "/clear") {
		t.Error("should contain /clear")
	}
	if !strings.Contains(stripped, "/help") {
		t.Error("should contain /help")
	}
	if !strings.Contains(stripped, "/exit") {
		t.Error("should contain /exit")
	}
	if !strings.Contains(stripped, "Clear conversation") {
		t.Error("should contain description")
	}
}

func TestCommandPalette_filter_narrows_results(t *testing.T) {
	t.Parallel()

	cmds := []CommandEntry{
		{Name: "clear", Description: "Clear conversation"},
		{Name: "compact", Description: "Compact history"},
		{Name: "help", Description: "Show help"},
	}
	cp := NewCommandPalette(cmds)
	cp.SetFilter("cl")

	buf := tui.AcquireBuffer()
	defer tui.ReleaseBuffer(buf)
	cp.Render(buf, 60)

	output := strings.Join(buf.Lines, "\n")
	stripped := width.StripANSI(output)

	if !strings.Contains(stripped, "/clear") {
		t.Error("should contain /clear (matches 'cl')")
	}
	if strings.Contains(stripped, "/help") {
		t.Error("should NOT contain /help (doesn't match 'cl')")
	}
}

func TestCommandPalette_selected_returns_command(t *testing.T) {
	t.Parallel()

	cmds := []CommandEntry{
		{Name: "clear", Description: "Clear conversation"},
		{Name: "help", Description: "Show help"},
	}
	cp := NewCommandPalette(cmds)

	// Default selection is index 0
	selected := cp.Selected()
	if selected != "clear" {
		t.Errorf("default selection should be 'clear', got %q", selected)
	}

	// Move down
	cp.MoveDown()
	selected = cp.Selected()
	if selected != "help" {
		t.Errorf("after MoveDown, selection should be 'help', got %q", selected)
	}

	// Move down again wraps to top
	cp.MoveDown()
	selected = cp.Selected()
	if selected != "clear" {
		t.Errorf("after wrapping, selection should be 'clear', got %q", selected)
	}
}

func TestCommandPalette_empty_filter_shows_all(t *testing.T) {
	t.Parallel()

	cmds := []CommandEntry{
		{Name: "a", Description: "aaa"},
		{Name: "b", Description: "bbb"},
		{Name: "c", Description: "ccc"},
	}
	cp := NewCommandPalette(cmds)
	cp.SetFilter("")

	if cp.VisibleCount() != 3 {
		t.Errorf("empty filter should show all 3 commands, got %d", cp.VisibleCount())
	}
}

func TestCommandPalette_max_visible_items(t *testing.T) {
	t.Parallel()

	cmds := make([]CommandEntry, 30)
	for i := range cmds {
		cmds[i] = CommandEntry{Name: strings.Repeat("x", i+1), Description: "desc"}
	}
	cp := NewCommandPalette(cmds)

	buf := tui.AcquireBuffer()
	defer tui.ReleaseBuffer(buf)
	cp.Render(buf, 60)

	// Should cap visible items to maxVisibleItems (10)
	if len(buf.Lines) > maxVisibleItems+1 { // +1 for possible header
		t.Errorf("should render at most %d+1 lines, got %d", maxVisibleItems, len(buf.Lines))
	}
}
