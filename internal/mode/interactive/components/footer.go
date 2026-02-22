// ABOUTME: Information-dense two-line status bar with project, git, model, cost, permissions
// ABOUTME: Line 1: path + branch + model + cost; Line 2: mode + context% + stats

package components

import (
	"fmt"
	"strings"
	"sync"

	"github.com/mauromedda/pi-coding-agent-go/internal/config"
	"github.com/mauromedda/pi-coding-agent-go/pkg/tui"
	"github.com/mauromedda/pi-coding-agent-go/pkg/tui/theme"
	"github.com/mauromedda/pi-coding-agent-go/pkg/tui/width"
)

// Footer displays status information at the bottom of the screen.
// It renders two lines with rich context about the current session.
type Footer struct {
	line1          string
	line2Left      string
	line2Right     string
	modeLabel      string
	contextPct     int
	thinking       config.ThinkingLevel
	model          string
	gitBranch      string
	cost           float64
	permissionMode string
	queuedCount    int
	mu             sync.Mutex
}

// NewFooter creates a Footer component.
func NewFooter() *Footer {
	return &Footer{
		thinking: config.ThinkingOff,
	}
}

// SetLine1 sets the first line content (e.g. project path, branch).
func (f *Footer) SetLine1(s string) {
	f.mu.Lock()
	f.line1 = s
	f.mu.Unlock()
}

// SetLine2 sets the second line with left-aligned and right-aligned parts.
func (f *Footer) SetLine2(left, right string) {
	f.mu.Lock()
	f.line2Left = left
	f.line2Right = right
	f.mu.Unlock()
}

// SetContent updates the footer text (backward compatibility).
// Sets line1 to the given content and clears line2.
func (f *Footer) SetContent(content string) {
	f.mu.Lock()
	f.line1 = content
	f.line2Left = ""
	f.line2Right = ""
	f.mu.Unlock()
}

// Content returns the current line1 text (backward compatibility).
func (f *Footer) Content() string {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.line1
}

// SetThinkingLevel sets the thinking level for display.
func (f *Footer) SetThinkingLevel(level config.ThinkingLevel) {
	f.mu.Lock()
	f.thinking = level
	f.mu.Unlock()
}

// SetModel sets the model name for display.
func (f *Footer) SetModel(name string) {
	f.mu.Lock()
	f.model = name
	f.mu.Unlock()
}

// SetContextPct sets the context window occupation percentage for display.
func (f *Footer) SetContextPct(pct int) {
	f.mu.Lock()
	f.contextPct = pct
	f.mu.Unlock()
}

// SetModeLabel sets the Plan/Edit mode indicator for display.
func (f *Footer) SetModeLabel(label string) {
	f.mu.Lock()
	f.modeLabel = label
	f.mu.Unlock()
}

// SetGitBranch sets the git branch name for display.
func (f *Footer) SetGitBranch(branch string) {
	f.mu.Lock()
	f.gitBranch = branch
	f.mu.Unlock()
}

// SetCost sets the running API cost for display.
func (f *Footer) SetCost(cost float64) {
	f.mu.Lock()
	f.cost = cost
	f.mu.Unlock()
}

// SetPermissionMode sets the permission mode label for display.
func (f *Footer) SetPermissionMode(mode string) {
	f.mu.Lock()
	f.permissionMode = mode
	f.mu.Unlock()
}

// SetQueuedCount sets the number of queued follow-up messages for display.
func (f *Footer) SetQueuedCount(n int) {
	f.mu.Lock()
	f.queuedCount = n
	f.mu.Unlock()
}

// Render writes the two footer lines with rich status information.
func (f *Footer) Render(out *tui.RenderBuffer, w int) {
	f.mu.Lock()
	line1 := f.line1
	line2Left := f.line2Left
	line2Right := f.line2Right
	modeLabel := f.modeLabel
	contextPct := f.contextPct
	thinking := f.thinking
	model := f.model
	gitBranch := f.gitBranch
	cost := f.cost
	permMode := f.permissionMode
	queued := f.queuedCount
	f.mu.Unlock()

	// === Line 1: project path + git branch + model + cost ===
	p := theme.Current().Palette
	var parts []string

	// Project path (from line1 or fallback)
	if line1 != "" {
		parts = append(parts, p.FooterPath.Apply(line1))
	}

	// Git branch with icon
	if gitBranch != "" {
		parts = append(parts, p.FooterBranch.Apply("\ue0a0 "+gitBranch))
	}

	// Model name
	if model != "" {
		parts = append(parts, p.FooterModel.Apply(model))
	}

	// Cost (only if > 0)
	if cost > 0 {
		parts = append(parts, p.FooterCost.Apply(fmt.Sprintf("$%.2f", cost)))
	}

	line1Str := strings.Join(parts, p.Muted.Apply("  "))
	out.WriteLine(line1Str)

	// === Line 2: mode + permissions + stats + context ===
	line2 := line2Left

	// Permission mode indicator
	if permMode != "" {
		permColor := p.Warning // yellow default
		switch strings.ToLower(permMode) {
		case "bypass", "yolo":
			permColor = p.Error // red for bypass
		case "normal", "plan":
			permColor = p.FooterPerm // green for safe modes
		}
		line2 += " " + permColor.Apply("▸▸ "+permMode)
	}

	if modeLabel != "" {
		line2 += " " + p.Warning.Apply(modeLabel)
	}

	if contextPct > 0 {
		ctxColor := p.Secondary // dim for < 60%
		if contextPct >= 80 {
			ctxColor = p.Error // red
		} else if contextPct >= 60 {
			ctxColor = p.Warning // yellow
		}
		line2 += " " + ctxColor.Apply(fmt.Sprintf("ctx %d%%", contextPct))
	}

	if queued > 0 {
		line2 += " " + p.Warning.Apply(fmt.Sprintf("[%d queued]", queued))
	}

	if thinking != config.ThinkingOff {
		line2 += " " + p.Info.Apply("["+thinking.String()+"]")
	}

	leftW := width.VisibleWidth(line2)
	rightW := width.VisibleWidth(line2Right)
	totalUsed := leftW + rightW
	var fullLine2 string
	if totalUsed >= w {
		// Narrow terminal: truncate left side, drop right
		fullLine2 = width.TruncateToWidth(line2, w)
	} else {
		pad := w - leftW - rightW
		fullLine2 = line2 + strings.Repeat(" ", pad) + p.Muted.Apply(line2Right)
	}
	out.WriteLine(fullLine2)
}

// Invalidate is a no-op for Footer.
func (f *Footer) Invalidate() {}
