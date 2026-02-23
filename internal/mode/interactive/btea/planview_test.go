// ABOUTME: Tests for PlanViewModel: approve/reject keys, scrolling, and view rendering
// ABOUTME: Table-driven tests verifying plan review overlay behavior

package btea

import (
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
)

// Compile-time checks: PlanViewModel and result messages must satisfy tea interfaces.
var (
	_ tea.Model = PlanViewModel{}
	_ tea.Msg   = PlanApprovedMsg{}
	_ tea.Msg   = PlanRejectedMsg{}
)

func TestPlanViewModel_Init(t *testing.T) {
	t.Parallel()
	m := NewPlanViewModel("test plan")
	if cmd := m.Init(); cmd != nil {
		t.Errorf("Init() returned non-nil cmd")
	}
}

func TestPlanViewModel_ApproveKey(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name string
		key  string
	}{
		{"y key", "y"},
		{"enter key", "enter"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			m := NewPlanViewModel("some plan")
			var keyMsg tea.KeyMsg
			if tt.key == "enter" {
				keyMsg = tea.KeyMsg{Type: tea.KeyEnter}
			} else {
				keyMsg = tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune(tt.key)}
			}
			_, cmd := m.Update(keyMsg)
			if cmd == nil {
				t.Fatal("cmd = nil; want PlanApprovedMsg cmd")
			}
			msg := cmd()
			if _, ok := msg.(PlanApprovedMsg); !ok {
				t.Errorf("cmd() = %T; want PlanApprovedMsg", msg)
			}
		})
	}
}

func TestPlanViewModel_RejectKey(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name string
		key  tea.KeyMsg
	}{
		{"n key", tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'n'}}},
		{"esc key", tea.KeyMsg{Type: tea.KeyEscape}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			m := NewPlanViewModel("some plan")
			_, cmd := m.Update(tt.key)
			if cmd == nil {
				t.Fatal("cmd = nil; want PlanRejectedMsg cmd")
			}
			msg := cmd()
			if _, ok := msg.(PlanRejectedMsg); !ok {
				t.Errorf("cmd() = %T; want PlanRejectedMsg", msg)
			}
		})
	}
}

func TestPlanViewModel_Scroll(t *testing.T) {
	t.Parallel()
	m := NewPlanViewModel("long plan content")

	// Scroll down
	downKeys := []tea.KeyMsg{
		{Type: tea.KeyRunes, Runes: []rune{'j'}},
		{Type: tea.KeyRunes, Runes: []rune{'j'}},
	}
	for _, k := range downKeys {
		updated, _ := m.Update(k)
		m = updated.(PlanViewModel)
	}
	if m.scroll != 2 {
		t.Errorf("scroll = %d; want 2 after 2 down presses", m.scroll)
	}

	// Scroll up
	upKey := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'k'}}
	updated, _ := m.Update(upKey)
	m = updated.(PlanViewModel)
	if m.scroll != 1 {
		t.Errorf("scroll = %d; want 1 after up press", m.scroll)
	}

	// Scroll up at 0 should stay at 0
	updated, _ = m.Update(upKey)
	m = updated.(PlanViewModel)
	updated, _ = m.Update(upKey)
	m = updated.(PlanViewModel)
	if m.scroll != 0 {
		t.Errorf("scroll = %d; want 0 (should not go negative)", m.scroll)
	}
}

func TestPlanViewModel_ScrollDownArrow(t *testing.T) {
	t.Parallel()
	m := NewPlanViewModel("plan")
	down := tea.KeyMsg{Type: tea.KeyDown}
	updated, _ := m.Update(down)
	m = updated.(PlanViewModel)
	if m.scroll != 1 {
		t.Errorf("scroll = %d; want 1 after down arrow", m.scroll)
	}
}

func TestPlanViewModel_ScrollUpArrow(t *testing.T) {
	t.Parallel()
	m := NewPlanViewModel("plan")
	// First scroll down, then up
	down := tea.KeyMsg{Type: tea.KeyDown}
	up := tea.KeyMsg{Type: tea.KeyUp}
	updated, _ := m.Update(down)
	m = updated.(PlanViewModel)
	updated, _ = m.Update(up)
	m = updated.(PlanViewModel)
	if m.scroll != 0 {
		t.Errorf("scroll = %d; want 0 after up arrow", m.scroll)
	}
}

func TestPlanViewModel_View_ContainsPlan(t *testing.T) {
	t.Parallel()
	m := NewPlanViewModel("This is the generated plan content")
	m.width = 80
	m.height = 24
	view := m.View()

	if !strings.Contains(view, "This is the generated plan content") {
		t.Errorf("View() missing plan content; got %q", view)
	}
}

func TestPlanViewModel_View_ContainsHeader(t *testing.T) {
	t.Parallel()
	m := NewPlanViewModel("plan")
	m.width = 80
	m.height = 24
	view := m.View()

	if !strings.Contains(view, "Plan Review") {
		t.Errorf("View() missing header 'Plan Review'; got %q", view)
	}
}

func TestPlanViewModel_View_ContainsKeybindings(t *testing.T) {
	t.Parallel()
	m := NewPlanViewModel("plan")
	m.width = 80
	m.height = 24
	view := m.View()

	if !strings.Contains(view, "y=approve") {
		t.Errorf("View() missing keybinding hint 'y=approve'; got %q", view)
	}
	if !strings.Contains(view, "n=reject") {
		t.Errorf("View() missing keybinding hint 'n=reject'; got %q", view)
	}
}

func TestPlanViewModel_WindowSizeMsg(t *testing.T) {
	t.Parallel()
	m := NewPlanViewModel("plan")
	updated, _ := m.Update(tea.WindowSizeMsg{Width: 120, Height: 40})
	pm := updated.(PlanViewModel)
	if pm.width != 120 {
		t.Errorf("width = %d; want 120", pm.width)
	}
	if pm.height != 40 {
		t.Errorf("height = %d; want 40", pm.height)
	}
}

// --- WS2: Overlay border tests ---

func TestPlanViewModel_ViewHasBorder(t *testing.T) {
	t.Parallel()
	m := NewPlanViewModel("test plan content")
	m.width = 80
	m.height = 24
	view := m.View()

	if !strings.Contains(view, "╭") || !strings.Contains(view, "╰") {
		t.Errorf("View() should contain rounded border chars (╭/╰); got:\n%s", view)
	}
}

func TestPlanViewModel_ViewHasScrollIndicator(t *testing.T) {
	t.Parallel()
	// Create a plan with many lines
	lines := make([]string, 50)
	for i := range lines {
		lines[i] = "Line content"
	}
	m := NewPlanViewModel(strings.Join(lines, "\n"))
	m.width = 80
	m.height = 20
	view := m.View()

	if !strings.Contains(view, "lines") {
		t.Errorf("View() should contain scroll indicator with 'lines'; got:\n%s", view)
	}
}

func TestOverlayRender_CentersOverlay(t *testing.T) {
	t.Parallel()
	bg := "background line 1\nbackground line 2\nbackground line 3\nbackground line 4\nbackground line 5"
	overlay := "OVR"

	result := overlayRender(bg, overlay, 20, 5)

	// Result should have 5 lines (matching height)
	resultLines := strings.Split(result, "\n")
	if len(resultLines) != 5 {
		t.Errorf("overlayRender produced %d lines; want 5", len(resultLines))
	}

	// The overlay text should appear somewhere in the output
	if !strings.Contains(result, "OVR") {
		t.Errorf("overlayRender output should contain overlay text; got:\n%s", result)
	}
}

func TestOverlayRender_PreservesBackgroundOnNonOverlayRows(t *testing.T) {
	t.Parallel()
	bg := "AAAAAA\nBBBBBB\nCCCCCC\nDDDDDD\nEEEEEE"
	overlay := "X"

	result := overlayRender(bg, overlay, 6, 5)

	// At least some background rows should be preserved (not all replaced)
	lines := strings.Split(result, "\n")
	hasBackground := false
	for _, l := range lines {
		if strings.Contains(l, "AAAAAA") || strings.Contains(l, "EEEEE") {
			hasBackground = true
			break
		}
	}
	if !hasBackground {
		t.Errorf("overlayRender should preserve some background rows; got:\n%s", result)
	}
}
