// ABOUTME: Tests for CostViewModel: dismiss keys, token display, and budget rendering
// ABOUTME: Table-driven tests verifying cost dashboard overlay behavior

package btea

import (
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
)

// Compile-time check: CostViewModel must satisfy tea.Model.
var _ tea.Model = CostViewModel{}

func TestCostViewModel_Init(t *testing.T) {
	t.Parallel()
	m := NewCostViewModel(100, 50, 3, 0.42, 10.0, 4.2)
	if cmd := m.Init(); cmd != nil {
		t.Errorf("Init() returned non-nil cmd")
	}
}

func TestCostViewModel_DismissEsc(t *testing.T) {
	t.Parallel()
	m := NewCostViewModel(100, 50, 3, 0.42, 10.0, 4.2)
	key := tea.KeyMsg{Type: tea.KeyEscape}
	_, cmd := m.Update(key)
	if cmd == nil {
		t.Fatal("cmd = nil; want DismissOverlayMsg cmd")
	}
	msg := cmd()
	if _, ok := msg.(DismissOverlayMsg); !ok {
		t.Errorf("cmd() = %T; want DismissOverlayMsg", msg)
	}
}

func TestCostViewModel_DismissCtrlT(t *testing.T) {
	t.Parallel()
	m := NewCostViewModel(100, 50, 3, 0.42, 10.0, 4.2)
	key := tea.KeyMsg{Type: tea.KeyCtrlT}
	_, cmd := m.Update(key)
	if cmd == nil {
		t.Fatal("cmd = nil; want DismissOverlayMsg cmd")
	}
	msg := cmd()
	if _, ok := msg.(DismissOverlayMsg); !ok {
		t.Errorf("cmd() = %T; want DismissOverlayMsg", msg)
	}
}

func TestCostViewModel_DismissQ(t *testing.T) {
	t.Parallel()
	m := NewCostViewModel(100, 50, 3, 0.42, 10.0, 4.2)
	key := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'q'}}
	_, cmd := m.Update(key)
	if cmd == nil {
		t.Fatal("cmd = nil; want DismissOverlayMsg cmd")
	}
	msg := cmd()
	if _, ok := msg.(DismissOverlayMsg); !ok {
		t.Errorf("cmd() = %T; want DismissOverlayMsg", msg)
	}
}

func TestCostViewModel_View_ShowsTokens(t *testing.T) {
	t.Parallel()
	m := NewCostViewModel(12345, 6789, 5, 0.42, 10.0, 4.2)
	m.width = 80
	m.height = 24
	view := m.View()

	if !strings.Contains(view, "12,345") {
		t.Errorf("View() missing input tokens '12,345'; got %q", view)
	}
	if !strings.Contains(view, "6,789") {
		t.Errorf("View() missing output tokens '6,789'; got %q", view)
	}
}

func TestCostViewModel_View_ShowsCost(t *testing.T) {
	t.Parallel()
	m := NewCostViewModel(100, 50, 5, 0.42, 10.0, 4.2)
	m.width = 80
	m.height = 24
	view := m.View()

	if !strings.Contains(view, "$0.42") {
		t.Errorf("View() missing cost '$0.42'; got %q", view)
	}
}

func TestCostViewModel_View_ShowsBudget(t *testing.T) {
	t.Parallel()
	m := NewCostViewModel(100, 50, 5, 0.42, 10.0, 4.2)
	m.width = 80
	m.height = 24
	view := m.View()

	if !strings.Contains(view, "$10.00") {
		t.Errorf("View() missing budget '$10.00'; got %q", view)
	}
	if !strings.Contains(view, "4.2%") {
		t.Errorf("View() missing budget used '4.2%%'; got %q", view)
	}
}

func TestCostViewModel_View_ShowsCallCount(t *testing.T) {
	t.Parallel()
	m := NewCostViewModel(100, 50, 5, 0.42, 10.0, 4.2)
	m.width = 80
	m.height = 24
	view := m.View()

	if !strings.Contains(view, "5") {
		t.Errorf("View() missing call count '5'; got %q", view)
	}
}

func TestCostViewModel_View_ShowsHeader(t *testing.T) {
	t.Parallel()
	m := NewCostViewModel(100, 50, 5, 0.42, 10.0, 4.2)
	m.width = 80
	m.height = 24
	view := m.View()

	if !strings.Contains(view, "Token & Cost Dashboard") {
		t.Errorf("View() missing header 'Token & Cost Dashboard'; got %q", view)
	}
}

func TestCostViewModel_View_ShowsDismissHint(t *testing.T) {
	t.Parallel()
	m := NewCostViewModel(100, 50, 5, 0.42, 10.0, 4.2)
	m.width = 80
	m.height = 24
	view := m.View()

	if !strings.Contains(view, "ESC") {
		t.Errorf("View() missing dismiss hint 'ESC'; got %q", view)
	}
}

func TestCostViewModel_WindowSizeMsg(t *testing.T) {
	t.Parallel()
	m := NewCostViewModel(100, 50, 3, 0.42, 10.0, 4.2)
	updated, _ := m.Update(tea.WindowSizeMsg{Width: 120, Height: 40})
	cm := updated.(CostViewModel)
	if cm.width != 120 {
		t.Errorf("width = %d; want 120", cm.width)
	}
	if cm.height != 40 {
		t.Errorf("height = %d; want 40", cm.height)
	}
}
