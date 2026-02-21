// ABOUTME: Interactive TUI mode: main loop, mode switching (Plan/Edit)
// ABOUTME: Orchestrates agent session, TUI rendering, and keyboard shortcuts

package interactive

import (
	"context"
	"fmt"

	"github.com/mauromedda/pi-coding-agent-go/internal/permission"
	tuipkg "github.com/mauromedda/pi-coding-agent-go/pkg/tui"
)

// Mode represents the current editing mode.
type Mode int

const (
	ModePlan Mode = iota
	ModeEdit
)

// String returns the mode display name.
func (m Mode) String() string {
	switch m {
	case ModePlan:
		return "PLAN"
	case ModeEdit:
		return "EDIT"
	default:
		return "UNKNOWN"
	}
}

// App is the main interactive application.
type App struct {
	tui      *tuipkg.TUI
	mode     Mode
	checker  *permission.Checker
	ctx      context.Context
	cancelFn context.CancelFunc
}

// New creates a new interactive app.
func New(writer tuipkg.Writer, width, height int, checker *permission.Checker) *App {
	ctx, cancel := context.WithCancel(context.Background())
	app := &App{
		tui:      tuipkg.New(writer, width, height),
		mode:     ModePlan,
		checker:  checker,
		ctx:      ctx,
		cancelFn: cancel,
	}
	return app
}

// ToggleMode switches between Plan and Edit modes.
func (a *App) ToggleMode() {
	switch a.mode {
	case ModePlan:
		a.mode = ModeEdit
		a.checker.SetMode(permission.ModeNormal)
	case ModeEdit:
		a.mode = ModePlan
		a.checker.SetMode(permission.ModePlan)
	}
}

// Mode returns the current mode.
func (a *App) Mode() Mode {
	return a.mode
}

// ModeLabel returns the display label for the footer.
func (a *App) ModeLabel() string {
	switch a.mode {
	case ModePlan:
		return "[PLAN] Shift+Tab -> Edit"
	case ModeEdit:
		return "[EDIT] Shift+Tab -> Plan"
	default:
		return ""
	}
}

// TUI returns the underlying TUI engine.
func (a *App) TUI() *tuipkg.TUI {
	return a.tui
}

// Start begins the TUI render loop.
func (a *App) Start() {
	a.tui.Start()
}

// Stop shuts down the TUI and cancels context.
func (a *App) Stop() {
	a.cancelFn()
	a.tui.Stop()
}

// Context returns the app's context.
func (a *App) Context() context.Context {
	return a.ctx
}

// SetYoloMode enables yolo mode (skip all permission prompts).
func (a *App) SetYoloMode() {
	a.mode = ModeEdit
	a.checker.SetMode(permission.ModeYolo)
}

// StatusLine returns the status information for the footer.
func (a *App) StatusLine(model string, totalCost float64) string {
	return fmt.Sprintf("%s | %s | $%.4f", a.ModeLabel(), model, totalCost)
}
