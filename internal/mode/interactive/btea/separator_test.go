// ABOUTME: Tests for SeparatorModel Bubble Tea leaf component
// ABOUTME: Verifies tea.Model interface, View output, and WindowSizeMsg handling

package btea

import (
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
)

// Compile-time check: SeparatorModel must satisfy tea.Model.
var _ tea.Model = SeparatorModel{}

func TestSeparatorModel_Init(t *testing.T) {
	m := NewSeparatorModel()
	if cmd := m.Init(); cmd != nil {
		t.Errorf("Init() returned non-nil cmd")
	}
}

func TestSeparatorModel_View(t *testing.T) {
	tests := []struct {
		name      string
		width     int
		wantChar  string
		wantCount int
	}{
		{"standard width", 40, "─", 40},
		{"narrow width", 5, "─", 5},
		{"zero width", 0, "", 0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := NewSeparatorModel()
			updated, _ := m.Update(tea.WindowSizeMsg{Width: tt.width, Height: 24})
			view := updated.(SeparatorModel).View()

			if tt.wantCount == 0 {
				if view != "" {
					t.Errorf("View() = %q; want empty", view)
				}
				return
			}

			if !strings.Contains(view, tt.wantChar) {
				t.Errorf("View() missing %q character", tt.wantChar)
			}

			count := strings.Count(view, tt.wantChar)
			if count != tt.wantCount {
				t.Errorf("View() has %d '─' chars; want %d", count, tt.wantCount)
			}
		})
	}
}

func TestSeparatorModel_WindowSizeMsg(t *testing.T) {
	m := NewSeparatorModel()
	updated, cmd := m.Update(tea.WindowSizeMsg{Width: 80, Height: 24})
	if cmd != nil {
		t.Errorf("Update(WindowSizeMsg) returned non-nil cmd")
	}
	sep := updated.(SeparatorModel)
	if sep.width != 80 {
		t.Errorf("width = %d; want 80", sep.width)
	}
}
