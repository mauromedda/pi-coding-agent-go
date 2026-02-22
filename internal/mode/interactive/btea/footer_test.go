// ABOUTME: Tests for FooterModel Bubble Tea leaf component
// ABOUTME: Verifies two-line status bar rendering, WithXxx builders, and AgentUsageMsg

package btea

import (
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/mauromedda/pi-coding-agent-go/internal/config"
	"github.com/mauromedda/pi-coding-agent-go/pkg/ai"
)

// Compile-time check: FooterModel must satisfy tea.Model.
var _ tea.Model = FooterModel{}

func TestFooterModel_Init(t *testing.T) {
	m := NewFooterModel()
	if cmd := m.Init(); cmd != nil {
		t.Errorf("Init() returned non-nil cmd")
	}
}

func TestFooterModel_WithMethods(t *testing.T) {
	m := NewFooterModel()
	m = m.WithPath("/home/test")
	m = m.WithGitBranch("main")
	m = m.WithModel("claude-opus-4")
	m = m.WithCost(1.23)
	m = m.WithModeLabel("Plan")
	m = m.WithContextPct(45)
	m = m.WithThinking(config.ThinkingMedium)
	m = m.WithPermissionMode("normal")
	m = m.WithQueuedCount(3)

	if m.path != "/home/test" {
		t.Errorf("path = %q; want /home/test", m.path)
	}
	if m.gitBranch != "main" {
		t.Errorf("gitBranch = %q; want main", m.gitBranch)
	}
	if m.model != "claude-opus-4" {
		t.Errorf("model = %q; want claude-opus-4", m.model)
	}
	if m.cost != 1.23 {
		t.Errorf("cost = %f; want 1.23", m.cost)
	}
	if m.modeLabel != "Plan" {
		t.Errorf("modeLabel = %q; want Plan", m.modeLabel)
	}
	if m.contextPct != 45 {
		t.Errorf("contextPct = %d; want 45", m.contextPct)
	}
	if m.thinking != config.ThinkingMedium {
		t.Errorf("thinking = %v; want ThinkingMedium", m.thinking)
	}
	if m.permissionMode != "normal" {
		t.Errorf("permissionMode = %q; want normal", m.permissionMode)
	}
	if m.queuedCount != 3 {
		t.Errorf("queuedCount = %d; want 3", m.queuedCount)
	}
}

func TestFooterModel_ViewContainsModelName(t *testing.T) {
	m := NewFooterModel()
	m = m.WithModel("claude-opus-4")
	m.width = 80
	view := m.View()
	if !strings.Contains(view, "claude-opus-4") {
		t.Errorf("View() missing model name; got %q", view)
	}
}

func TestFooterModel_ViewContainsPath(t *testing.T) {
	m := NewFooterModel()
	m = m.WithPath("/home/test/project")
	m.width = 80
	view := m.View()
	if !strings.Contains(view, "/home/test/project") {
		t.Errorf("View() missing path; got %q", view)
	}
}

func TestFooterModel_ViewContainsBranch(t *testing.T) {
	m := NewFooterModel()
	m = m.WithGitBranch("feature-x")
	m.width = 80
	view := m.View()
	if !strings.Contains(view, "feature-x") {
		t.Errorf("View() missing branch; got %q", view)
	}
}

func TestFooterModel_ViewContainsCost(t *testing.T) {
	m := NewFooterModel()
	m = m.WithCost(0.42)
	m.width = 80
	view := m.View()
	if !strings.Contains(view, "$0.42") {
		t.Errorf("View() missing cost; got %q", view)
	}
}

func TestFooterModel_ViewContainsModeLabel(t *testing.T) {
	m := NewFooterModel()
	m = m.WithModeLabel("Plan")
	m.width = 80
	view := m.View()
	if !strings.Contains(view, "Plan") {
		t.Errorf("View() missing mode label; got %q", view)
	}
}

func TestFooterModel_ViewContainsContextPct(t *testing.T) {
	m := NewFooterModel()
	m = m.WithContextPct(75)
	m.width = 80
	view := m.View()
	if !strings.Contains(view, "ctx 75%") {
		t.Errorf("View() missing context pct; got %q", view)
	}
}

func TestFooterModel_ViewContainsQueued(t *testing.T) {
	m := NewFooterModel()
	m = m.WithQueuedCount(5)
	m.width = 80
	view := m.View()
	if !strings.Contains(view, "5 queued") {
		t.Errorf("View() missing queued count; got %q", view)
	}
}

func TestFooterModel_ViewContainsThinking(t *testing.T) {
	m := NewFooterModel()
	m = m.WithThinking(config.ThinkingHigh)
	m.width = 80
	view := m.View()
	if !strings.Contains(view, "high") {
		t.Errorf("View() missing thinking level; got %q", view)
	}
}

func TestFooterModel_ViewHidesCostWhenZero(t *testing.T) {
	m := NewFooterModel()
	m = m.WithCost(0)
	m.width = 80
	view := m.View()
	if strings.Contains(view, "$") {
		t.Errorf("View() should hide cost when zero; got %q", view)
	}
}

func TestFooterModel_ViewHidesQueuedWhenZero(t *testing.T) {
	m := NewFooterModel()
	m = m.WithQueuedCount(0)
	m.width = 80
	view := m.View()
	if strings.Contains(view, "queued") {
		t.Errorf("View() should hide queued when zero; got %q", view)
	}
}

func TestFooterModel_AgentUsageMsg(t *testing.T) {
	m := NewFooterModel()
	m.width = 80
	usage := &ai.Usage{InputTokens: 1000, OutputTokens: 500}
	updated, cmd := m.Update(AgentUsageMsg{Usage: usage})
	if cmd != nil {
		t.Errorf("Update(AgentUsageMsg) returned non-nil cmd")
	}
	f := updated.(FooterModel)
	// Cost = 1000*3/1_000_000 + 500*15/1_000_000 = 0.003 + 0.0075 = 0.0105
	wantCost := 0.0105
	if f.cost < wantCost-0.001 || f.cost > wantCost+0.001 {
		t.Errorf("cost = %f; want ~%f", f.cost, wantCost)
	}
}

func TestFooterModel_WindowSizeMsg(t *testing.T) {
	m := NewFooterModel()
	updated, cmd := m.Update(tea.WindowSizeMsg{Width: 120, Height: 40})
	if cmd != nil {
		t.Errorf("Update(WindowSizeMsg) returned non-nil cmd")
	}
	f := updated.(FooterModel)
	if f.width != 120 {
		t.Errorf("width = %d; want 120", f.width)
	}
}

func TestFooterModel_ViewTwoLines(t *testing.T) {
	m := NewFooterModel()
	m = m.WithPath("/home")
	m = m.WithModel("model")
	m = m.WithModeLabel("Plan")
	m.width = 80

	view := m.View()
	lines := strings.Split(view, "\n")
	// Should produce exactly 2 lines (line1 + line2)
	if len(lines) != 2 {
		t.Errorf("View() produced %d lines; want 2; view=%q", len(lines), view)
	}
}

func TestFooterModel_ViewPermissionMode(t *testing.T) {
	tests := []struct {
		name string
		mode string
		want string
	}{
		{"normal mode", "normal", "normal"},
		{"bypass mode", "bypass", "bypass"},
		{"yolo mode", "yolo", "yolo"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := NewFooterModel()
			m = m.WithPermissionMode(tt.mode)
			m.width = 80
			view := m.View()
			if !strings.Contains(view, tt.want) {
				t.Errorf("View() missing permission mode %q; got %q", tt.want, view)
			}
		})
	}
}
