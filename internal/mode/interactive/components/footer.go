// ABOUTME: Information-dense two-line status bar with project, git, model, cost, permissions
// ABOUTME: Line 1: path + branch + model + cost; Line 2: mode + context% + stats

package components

import (
	"fmt"
	"strings"
	"sync"

	"github.com/mauromedda/pi-coding-agent-go/internal/config"
	"github.com/mauromedda/pi-coding-agent-go/pkg/tui"
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
	f.mu.Unlock()

	// === Line 1: project path + git branch + model + cost ===
	var parts []string

	// Project path (from line1 or fallback)
	if line1 != "" {
		parts = append(parts, fmt.Sprintf("\x1b[2;36m%s\x1b[0m", line1))
	}

	// Git branch with icon
	if gitBranch != "" {
		parts = append(parts, fmt.Sprintf("\x1b[2;32m\ue0a0 %s\x1b[0m", gitBranch))
	}

	// Model name
	if model != "" {
		parts = append(parts, fmt.Sprintf("\x1b[2;35m%s\x1b[0m", model))
	}

	// Cost (only if > 0)
	if cost > 0 {
		parts = append(parts, fmt.Sprintf("\x1b[2;33m$%.2f\x1b[0m", cost))
	}

	line1Str := strings.Join(parts, "\x1b[2m  \x1b[0m")
	out.WriteLine(line1Str)

	// === Line 2: mode + permissions + stats + context ===
	line2 := line2Left

	// Permission mode indicator
	if permMode != "" {
		permColor := "\x1b[33m" // yellow default
		switch strings.ToLower(permMode) {
		case "bypass", "yolo":
			permColor = "\x1b[31m" // red for bypass
		case "normal", "plan":
			permColor = "\x1b[32m" // green for safe modes
		}
		line2 += fmt.Sprintf(" %s▸▸ %s\x1b[0m", permColor, permMode)
	}

	if modeLabel != "" {
		line2 += fmt.Sprintf(" \x1b[33m%s\x1b[0m", modeLabel)
	}

	if contextPct > 0 {
		ctxColor := "\x1b[90m" // dim for < 60%
		if contextPct >= 80 {
			ctxColor = "\x1b[31m" // red
		} else if contextPct >= 60 {
			ctxColor = "\x1b[33m" // yellow
		}
		line2 += fmt.Sprintf(" %sctx %d%%\x1b[0m", ctxColor, contextPct)
	}

	if thinking != config.ThinkingOff {
		line2 += fmt.Sprintf(" \x1b[36m[%s]\x1b[0m", thinking.String())
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
		fullLine2 = line2 + strings.Repeat(" ", pad) + "\x1b[2m" + line2Right + "\x1b[0m"
	}
	out.WriteLine(fullLine2)
}

// Invalidate is a no-op for Footer.
func (f *Footer) Invalidate() {}
