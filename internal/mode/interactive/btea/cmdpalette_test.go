// ABOUTME: Tests for CmdPaletteModel Bubble Tea leaf component
// ABOUTME: Verifies filtering, wrapping navigation, selection/dismiss msgs, View rendering

package btea

import (
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
)

// Compile-time check: CmdPaletteModel must satisfy tea.Model.
var _ tea.Model = CmdPaletteModel{}

func TestCmdPaletteModel_Init(t *testing.T) {
	m := NewCmdPaletteModel(nil)
	if cmd := m.Init(); cmd != nil {
		t.Errorf("Init() returned non-nil cmd")
	}
}

func TestCmdPaletteModel_NewPopulatesVisible(t *testing.T) {
	cmds := []CommandEntry{
		{Name: "help", Description: "show help"},
		{Name: "quit", Description: "exit app"},
	}
	m := NewCmdPaletteModel(cmds)
	if len(m.visible) != 2 {
		t.Fatalf("visible len = %d; want 2", len(m.visible))
	}
}

func TestCmdPaletteModel_FilterReducesVisible(t *testing.T) {
	cmds := []CommandEntry{
		{Name: "help", Description: "show help"},
		{Name: "quit", Description: "exit app"},
		{Name: "history", Description: "show history"},
	}
	m := NewCmdPaletteModel(cmds)
	m = m.SetFilter("h")
	// "h" matches "help" and "history" via case-insensitive contains
	if len(m.visible) != 2 {
		t.Errorf("SetFilter('h'): visible len = %d; want 2", len(m.visible))
	}
}

func TestCmdPaletteModel_DownUpWrapping(t *testing.T) {
	cmds := []CommandEntry{
		{Name: "a"},
		{Name: "b"},
		{Name: "c"},
	}
	m := NewCmdPaletteModel(cmds)

	// Down to end
	updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyDown})
	m = updated.(CmdPaletteModel)
	if m.selected != 1 {
		t.Errorf("after down: selected = %d; want 1", m.selected)
	}

	updated, _ = m.Update(tea.KeyMsg{Type: tea.KeyDown})
	m = updated.(CmdPaletteModel)
	if m.selected != 2 {
		t.Errorf("after 2x down: selected = %d; want 2", m.selected)
	}

	// Wrap around
	updated, _ = m.Update(tea.KeyMsg{Type: tea.KeyDown})
	m = updated.(CmdPaletteModel)
	if m.selected != 0 {
		t.Errorf("after wrap-down: selected = %d; want 0", m.selected)
	}

	// Up wraps to end
	updated, _ = m.Update(tea.KeyMsg{Type: tea.KeyUp})
	m = updated.(CmdPaletteModel)
	if m.selected != 2 {
		t.Errorf("after wrap-up: selected = %d; want 2", m.selected)
	}
}

func TestCmdPaletteModel_Selected(t *testing.T) {
	cmds := []CommandEntry{
		{Name: "help"},
		{Name: "quit"},
	}
	m := NewCmdPaletteModel(cmds)
	if got := m.Selected(); got != "help" {
		t.Errorf("Selected() = %q; want %q", got, "help")
	}

	updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyDown})
	m = updated.(CmdPaletteModel)
	if got := m.Selected(); got != "quit" {
		t.Errorf("after down: Selected() = %q; want %q", got, "quit")
	}
}

func TestCmdPaletteModel_SelectedEmpty(t *testing.T) {
	m := NewCmdPaletteModel(nil)
	if got := m.Selected(); got != "" {
		t.Errorf("Selected() on empty = %q; want empty", got)
	}
}

func TestCmdPaletteModel_EnterReturnsSelectMsg(t *testing.T) {
	cmds := []CommandEntry{
		{Name: "help"},
	}
	m := NewCmdPaletteModel(cmds)
	_, cmd := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	if cmd == nil {
		t.Fatal("Enter did not produce a tea.Cmd")
	}
	msg := cmd()
	sel, ok := msg.(CmdPaletteSelectMsg)
	if !ok {
		t.Fatalf("cmd() returned %T; want CmdPaletteSelectMsg", msg)
	}
	if sel.Name != "help" {
		t.Errorf("CmdPaletteSelectMsg.Name = %q; want %q", sel.Name, "help")
	}
}

func TestCmdPaletteModel_EscReturnsDismissMsg(t *testing.T) {
	m := NewCmdPaletteModel([]CommandEntry{{Name: "help"}})
	_, cmd := m.Update(tea.KeyMsg{Type: tea.KeyEsc})
	if cmd == nil {
		t.Fatal("Esc did not produce a tea.Cmd")
	}
	msg := cmd()
	if _, ok := msg.(CmdPaletteDismissMsg); !ok {
		t.Errorf("cmd() returned %T; want CmdPaletteDismissMsg", msg)
	}
}

func TestCmdPaletteModel_ViewContainsCommandNames(t *testing.T) {
	cmds := []CommandEntry{
		{Name: "help", Description: "show help"},
		{Name: "quit", Description: "exit app"},
	}
	m := NewCmdPaletteModel(cmds)
	m.width = 80
	view := m.View()

	if !strings.Contains(view, "/help") {
		t.Errorf("View() missing '/help'")
	}
	if !strings.Contains(view, "/quit") {
		t.Errorf("View() missing '/quit'")
	}
}

func TestCmdPaletteModel_ViewEmptyList(t *testing.T) {
	m := NewCmdPaletteModel(nil)
	view := m.View()
	if view != "" {
		t.Errorf("View() on empty list = %q; want empty", view)
	}
}

func TestCmdPaletteModel_WindowSizeMsg(t *testing.T) {
	m := NewCmdPaletteModel(nil)
	updated, cmd := m.Update(tea.WindowSizeMsg{Width: 100, Height: 30})
	if cmd != nil {
		t.Errorf("Update(WindowSizeMsg) returned non-nil cmd")
	}
	w := updated.(CmdPaletteModel)
	if w.width != 100 {
		t.Errorf("width = %d; want 100", w.width)
	}
}

func TestCmdPaletteModel_MaxVisibleItemsCapped(t *testing.T) {
	cmds := make([]CommandEntry, 20)
	for i := range cmds {
		cmds[i] = CommandEntry{Name: "cmd"}
	}
	m := NewCmdPaletteModel(cmds)
	m.width = 80
	view := m.View()
	lines := strings.Split(view, "\n")
	// Remove trailing empty
	for len(lines) > 0 && lines[len(lines)-1] == "" {
		lines = lines[:len(lines)-1]
	}
	if len(lines) > maxCmdPaletteVisible {
		t.Errorf("View() rendered %d lines; want <= %d", len(lines), maxCmdPaletteVisible)
	}
}
