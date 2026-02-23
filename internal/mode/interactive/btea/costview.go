// ABOUTME: Token and cost dashboard overlay toggled with ctrl+t
// ABOUTME: Shows session totals, per-call breakdown, and budget status

package btea

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
)

// CostViewModel displays token usage and cost information.
type CostViewModel struct {
	totalInput    int
	totalOutput   int
	totalCost     float64
	callCount     int
	budgetUSD     float64
	budgetUsedPct float64
	width         int
	height        int
}

// NewCostViewModel creates a cost dashboard overlay.
func NewCostViewModel(input, output, calls int, cost, budget, budgetPct float64) CostViewModel {
	return CostViewModel{
		totalInput:    input,
		totalOutput:   output,
		callCount:     calls,
		totalCost:     cost,
		budgetUSD:     budget,
		budgetUsedPct: budgetPct,
	}
}

// Init returns nil; no startup commands needed.
func (m CostViewModel) Init() tea.Cmd { return nil }

// Update handles key events for the cost dashboard.
func (m CostViewModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "esc", "ctrl+t", "q":
			return m, func() tea.Msg { return DismissOverlayMsg{} }
		}
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
	}
	return m, nil
}

// View renders the cost dashboard.
func (m CostViewModel) View() string {
	var b strings.Builder
	b.WriteString("--- Token & Cost Dashboard ---\n")
	b.WriteString(fmt.Sprintf("  Input tokens:  %s\n", formatNumber(m.totalInput)))
	b.WriteString(fmt.Sprintf("  Output tokens: %s\n", formatNumber(m.totalOutput)))
	b.WriteString(fmt.Sprintf("  Total cost:    $%.2f\n", m.totalCost))
	b.WriteString(fmt.Sprintf("  API calls:     %d\n", m.callCount))
	b.WriteString(fmt.Sprintf("  Budget:        $%.2f\n", m.budgetUSD))
	b.WriteString(fmt.Sprintf("  Used:          %.1f%%\n", m.budgetUsedPct))
	b.WriteString("--- Press ESC to close ---")
	return b.String()
}

// formatNumber formats an integer with thousand separators.
func formatNumber(n int) string {
	s := fmt.Sprintf("%d", n)
	if len(s) <= 3 {
		return s
	}

	var parts []string
	for len(s) > 3 {
		parts = append([]string{s[len(s)-3:]}, parts...)
		s = s[:len(s)-3]
	}
	if len(s) > 0 {
		parts = append([]string{s}, parts...)
	}
	return strings.Join(parts, ",")
}
