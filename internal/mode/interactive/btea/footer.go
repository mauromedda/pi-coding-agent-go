// ABOUTME: FooterModel is a Bubble Tea leaf that renders a two-line status bar
// ABOUTME: Port of components/footer.go; shows path, branch, model, cost, mode, context%

package btea

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/mauromedda/pi-coding-agent-go/internal/config"
	"github.com/mauromedda/pi-coding-agent-go/pkg/tui/width"
)

// FooterModel renders a two-line status bar at the bottom of the terminal.
// Line 1: path + branch + model + cost.
// Line 2: mode + permissions + context% + queued + thinking.
type FooterModel struct {
	path           string
	gitBranch      string
	model          string
	cost           float64
	modeLabel      string
	contextPct     int
	thinking       config.ThinkingLevel
	permissionMode string
	queuedCount    int
	latencyClass   string
	width          int
}

// NewFooterModel creates an empty FooterModel.
func NewFooterModel() FooterModel {
	return FooterModel{
		thinking: config.ThinkingOff,
	}
}

// Init returns nil; no commands needed for a leaf model.
func (m FooterModel) Init() tea.Cmd {
	return nil
}

// Update handles messages relevant to the footer.
func (m FooterModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case AgentUsageMsg:
		if msg.Usage != nil {
			// Simple cost approximation: $3/M input + $15/M output
			m.cost += float64(msg.Usage.InputTokens)*3.0/1_000_000 +
				float64(msg.Usage.OutputTokens)*15.0/1_000_000
		}

	case tea.WindowSizeMsg:
		m.width = msg.Width
	}

	return m, nil
}

// WithPath returns a FooterModel with the path set.
func (m FooterModel) WithPath(p string) FooterModel {
	m.path = p
	return m
}

// WithGitBranch returns a FooterModel with the git branch set.
func (m FooterModel) WithGitBranch(b string) FooterModel {
	m.gitBranch = b
	return m
}

// WithModel returns a FooterModel with the model name set.
func (m FooterModel) WithModel(name string) FooterModel {
	m.model = name
	return m
}

// WithModeLabel returns a FooterModel with the mode label set.
func (m FooterModel) WithModeLabel(label string) FooterModel {
	m.modeLabel = label
	return m
}

// WithContextPct returns a FooterModel with the context percentage set.
func (m FooterModel) WithContextPct(pct int) FooterModel {
	m.contextPct = pct
	return m
}

// WithThinking returns a FooterModel with the thinking level set.
func (m FooterModel) WithThinking(level config.ThinkingLevel) FooterModel {
	m.thinking = level
	return m
}

// WithPermissionMode returns a FooterModel with the permission mode set.
func (m FooterModel) WithPermissionMode(mode string) FooterModel {
	m.permissionMode = mode
	return m
}

// WithQueuedCount returns a FooterModel with the queued count set.
func (m FooterModel) WithQueuedCount(n int) FooterModel {
	m.queuedCount = n
	return m
}

// WithCost returns a FooterModel with the cost set.
func (m FooterModel) WithCost(c float64) FooterModel {
	m.cost = c
	return m
}

// WithLatencyClass returns a FooterModel with the latency class indicator set.
func (m FooterModel) WithLatencyClass(class string) FooterModel {
	m.latencyClass = class
	return m
}

// View renders the two-line footer.
func (m FooterModel) View() string {
	s := Styles()

	// === Line 1: path + branch + model + cost ===
	var parts []string

	if m.path != "" {
		parts = append(parts, s.FooterPath.Render(m.path))
	}
	if m.gitBranch != "" {
		parts = append(parts, s.FooterBranch.Render("\ue0a0 "+m.gitBranch))
	}
	if m.model != "" {
		parts = append(parts, s.FooterModel.Render(m.model))
	}
	if m.latencyClass != "" {
		latencyStyle := s.Info
		switch m.latencyClass {
		case "local":
			latencyStyle = s.Success
		case "slow":
			latencyStyle = s.Warning
		}
		parts = append(parts, latencyStyle.Render("["+m.latencyClass+"]"))
	}
	if m.cost > 0 {
		parts = append(parts, s.FooterCost.Render(fmt.Sprintf("$%.2f", m.cost)))
	}

	line1 := strings.Join(parts, s.Muted.Render("  "))

	// === Line 2: mode + permissions + context% + queued + thinking ===
	var line2Parts []string

	if m.permissionMode != "" {
		permStyle := s.Warning
		switch strings.ToLower(m.permissionMode) {
		case "bypass", "yolo":
			permStyle = s.Error
		case "normal", "plan":
			permStyle = s.FooterPerm
		}
		line2Parts = append(line2Parts, permStyle.Render("▸▸ "+m.permissionMode))
	}

	if m.modeLabel != "" {
		line2Parts = append(line2Parts, s.Warning.Render(m.modeLabel))
	}

	if m.contextPct > 0 {
		ctxStyle := s.Secondary
		if m.contextPct >= 80 {
			ctxStyle = s.Error
		} else if m.contextPct >= 60 {
			ctxStyle = s.Warning
		}
		line2Parts = append(line2Parts, ctxStyle.Render(fmt.Sprintf("ctx %d%%", m.contextPct)))
	}

	if m.queuedCount > 0 {
		line2Parts = append(line2Parts, s.Warning.Render(fmt.Sprintf("[%d queued]", m.queuedCount)))
	}

	if m.thinking != config.ThinkingOff {
		line2Parts = append(line2Parts, s.Info.Render("["+m.thinking.String()+"]"))
	}

	line2 := strings.Join(line2Parts, " ")

	// Truncate if needed
	if m.width > 0 {
		line1W := width.VisibleWidth(line1)
		if line1W > m.width {
			line1 = width.TruncateToWidth(line1, m.width)
		}
		line2W := width.VisibleWidth(line2)
		if line2W > m.width {
			line2 = width.TruncateToWidth(line2, m.width)
		}
	}

	return line1 + "\n" + line2
}
