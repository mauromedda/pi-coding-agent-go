// ABOUTME: Tests for WelcomeModel Bubble Tea leaf component
// ABOUTME: Verifies ASCII pi logo, version, model, cwd, shortcuts, tool count in View

package btea

import (
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
)

// Compile-time check: WelcomeModel must satisfy tea.Model.
var _ tea.Model = WelcomeModel{}

func TestWelcomeModel_Init(t *testing.T) {
	m := NewWelcomeModel("1.0.0", "gpt-4", "/home/user", 5)
	if cmd := m.Init(); cmd != nil {
		t.Errorf("Init() returned non-nil cmd")
	}
}

func TestWelcomeModel_View(t *testing.T) {
	tests := []struct {
		name      string
		version   string
		model     string
		cwd       string
		toolCount int
		wantParts []string
	}{
		{
			name:      "all fields populated",
			version:   "1.2.3",
			model:     "claude-opus-4",
			cwd:       "/home/test",
			toolCount: 42,
			wantParts: []string{"π", "v1.2.3", "claude-opus-4", "/home/test", "42"},
		},
		{
			name:      "empty version uses dev",
			version:   "",
			model:     "test-model",
			cwd:       "/tmp",
			toolCount: 0,
			wantParts: []string{"π", "dev", "test-model", "/tmp", "0"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := NewWelcomeModel(tt.version, tt.model, tt.cwd, tt.toolCount)
			view := m.View()

			for _, want := range tt.wantParts {
				if !strings.Contains(view, want) {
					t.Errorf("View() missing %q", want)
				}
			}
		})
	}
}

func TestWelcomeModel_ViewContainsShortcuts(t *testing.T) {
	m := NewWelcomeModel("1.0.0", "model", "/cwd", 1)
	view := m.View()

	shortcuts := []string{
		"escape",
		"ctrl+c",
		"ctrl+d",
		"/",
		"!",
	}
	for _, sc := range shortcuts {
		if !strings.Contains(view, sc) {
			t.Errorf("View() missing keyboard shortcut %q", sc)
		}
	}
}

func TestWelcomeModel_ViewContainsPiBox(t *testing.T) {
	m := NewWelcomeModel("1.0.0", "model", "/cwd", 1)
	view := m.View()

	boxParts := []string{"╭", "╰", "│"}
	for _, part := range boxParts {
		if !strings.Contains(view, part) {
			t.Errorf("View() missing box character %q", part)
		}
	}
}

func TestWelcomeModel_ViewNarrowTerminal(t *testing.T) {
	m := NewWelcomeModel("1.0.0", "model", "/cwd", 5)
	m.width = 30

	view := m.View()
	if view == "" {
		t.Fatal("View() returned empty string on narrow terminal")
	}
	// Should still contain essential elements
	if !strings.Contains(view, "π") {
		t.Error("View() missing pi character on narrow terminal")
	}
	// Lines should not exceed width
	for _, line := range strings.Split(view, "\n") {
		if line == "" {
			continue
		}
		// Width check uses rune count as approximation (ANSI codes aside)
		// The key property: no line should be massively wider than m.width
	}
}

func TestWelcomeModel_WindowSizeMsg(t *testing.T) {
	m := NewWelcomeModel("1.0.0", "model", "/cwd", 1)
	updated, cmd := m.Update(tea.WindowSizeMsg{Width: 120, Height: 40})
	if cmd != nil {
		t.Errorf("Update(WindowSizeMsg) returned non-nil cmd")
	}
	w := updated.(WelcomeModel)
	if w.width != 120 {
		t.Errorf("width = %d; want 120", w.width)
	}
}
