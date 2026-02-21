// ABOUTME: Interactive TUI mode: main loop, mode switching (Plan/Edit), agent wiring
// ABOUTME: Orchestrates agent session, TUI rendering, keyboard input, and permission dialogs

package interactive

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync/atomic"
	"time"

	"github.com/mauromedda/pi-coding-agent-go/internal/agent"
	"github.com/mauromedda/pi-coding-agent-go/internal/commands"
	"github.com/mauromedda/pi-coding-agent-go/internal/mode/interactive/components"
	"github.com/mauromedda/pi-coding-agent-go/internal/permission"
	"github.com/mauromedda/pi-coding-agent-go/internal/statusline"
	"github.com/mauromedda/pi-coding-agent-go/pkg/ai"
	tuipkg "github.com/mauromedda/pi-coding-agent-go/pkg/tui"
	"github.com/mauromedda/pi-coding-agent-go/pkg/tui/component"
	"github.com/mauromedda/pi-coding-agent-go/pkg/tui/input"
	"github.com/mauromedda/pi-coding-agent-go/pkg/tui/key"
	"github.com/mauromedda/pi-coding-agent-go/pkg/tui/terminal"
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

// AppDeps bundles all dependencies for the interactive App.
type AppDeps struct {
	Terminal     terminal.Terminal
	Provider     ai.ApiProvider
	Model        *ai.Model
	Tools        []*agent.AgentTool
	Checker      *permission.Checker
	SystemPrompt string
	Version      string
	StatusEngine *statusline.Engine
}

// App is the main interactive application.
type App struct {
	tui      *tuipkg.TUI
	mode     Mode
	checker  *permission.Checker
	ctx      context.Context
	cancelFn context.CancelFunc

	// Dependencies
	term         terminal.Terminal
	provider     ai.ApiProvider
	model        *ai.Model
	tools        []*agent.AgentTool
	systemPrompt string
	version      string

	// TUI components
	editor  *component.Editor
	footer  *components.Footer
	current *components.AssistantMessage // current streaming response

	// State
	agentRunning atomic.Bool
	activeAgent  *agent.Agent
	activeDialog *components.PermissionDialog
	messages     []ai.Message // conversation history
	cmdRegistry  *commands.Registry

	// Token stats + context info
	totalInputTokens  atomic.Int64
	totalOutputTokens atomic.Int64
	lastResponseStart atomic.Int64 // UnixNano timestamp
	lastOutputTokens  atomic.Int64 // output tokens for current response
	tokPerSec         atomic.Int64 // tok/s × 10 (fixed-point)
	gitBranch         string

	// Saved permission mode for Plan/Edit toggle
	editPermMode permission.Mode

	// External status line engine (optional)
	statusEngine *statusline.Engine
}

// NewFromDeps creates a fully-wired interactive app from dependencies.
func NewFromDeps(deps AppDeps) *App {
	ctx, cancel := context.WithCancel(context.Background())
	app := &App{
		tui:          tuipkg.New(deps.Terminal, 80, 24),
		mode:         ModePlan,
		checker:      deps.Checker,
		ctx:          ctx,
		cancelFn:     cancel,
		term:         deps.Terminal,
		provider:     deps.Provider,
		model:        deps.Model,
		tools:        deps.Tools,
		systemPrompt: deps.SystemPrompt,
		version:      deps.Version,
		cmdRegistry:  commands.NewRegistry(),
		statusEngine: deps.StatusEngine,
	}
	return app
}

// New creates a new interactive app (backwards-compatible constructor).
func New(writer tuipkg.Writer, width, height int, checker *permission.Checker) *App {
	ctx, cancel := context.WithCancel(context.Background())
	return &App{
		tui:      tuipkg.New(writer, width, height),
		mode:     ModePlan,
		checker:  checker,
		ctx:      ctx,
		cancelFn: cancel,
	}
}

// Run is the blocking main loop. Enters raw mode, sets up components,
// reads input, and blocks until context is cancelled or user exits.
func (a *App) Run() error {
	if err := a.term.EnterRawMode(); err != nil {
		return fmt.Errorf("entering raw mode: %w", err)
	}
	defer func() { _ = a.term.ExitRawMode() }()

	// Get terminal size and configure TUI
	w, h, err := a.term.Size()
	if err != nil {
		w, h = 120, 40 // fallback
	}
	a.tui.SetSize(w, h)
	a.term.OnResize(func(width, height int) {
		a.tui.SetSize(width, height)
	})

	// Detect git branch
	a.gitBranch = detectGitBranch()

	// Set up editor and footer
	a.editor = component.NewEditor()
	a.editor.SetFocused(true)
	a.footer = components.NewFooter()
	a.updateFooter()

	container := a.tui.Container()
	container.Add(a.editor)
	container.Add(a.footer)

	// Start TUI render loop
	a.tui.Start()
	defer a.tui.Stop()

	// Print welcome line
	a.printWelcome(container)
	a.tui.RequestRender()

	// Wire permission checker ask function
	a.checker.SetAskFn(a.askPermission)

	// Start reading stdin keys
	stdinBuf := input.NewStdinBuffer(os.Stdin, a.onKey)
	go stdinBuf.Start(a.ctx)

	// Block until context done
	<-a.ctx.Done()
	return nil
}

// printWelcome adds a welcome message to the container.
func (a *App) printWelcome(container *tuipkg.Container) {
	ver := a.version
	if ver == "" {
		ver = "dev"
	}
	modelName := "none"
	if a.model != nil {
		modelName = a.model.Name
	}
	welcome := components.NewAssistantMessage()
	welcome.AppendText(fmt.Sprintf("pi-go %s | model: %s | mode: %s | tools: %d\nType your prompt and press Enter.\n",
		ver, modelName, a.mode, len(a.tools)))

	// Insert before editor (which is at index 0)
	container.Remove(a.editor)
	container.Remove(a.footer)
	container.Add(welcome)
	container.Add(a.editor)
	container.Add(a.footer)
}

// onKey handles a single key event from the StdinBuffer goroutine.
func (a *App) onKey(k key.Key) {
	// 1. Active overlay (permission dialog): route y/a/n/Esc
	if a.activeDialog != nil {
		switch k.Type {
		case key.KeyRune:
			switch k.Rune {
			case 'y', 'Y':
				a.activeDialog.Allow()
				a.tui.PopOverlay()
				a.activeDialog = nil
			case 'a', 'A':
				a.activeDialog.AllowAlways()
				a.tui.PopOverlay()
				a.activeDialog = nil
			case 'n', 'N':
				a.activeDialog.Deny()
				a.tui.PopOverlay()
				a.activeDialog = nil
			}
		case key.KeyEscape:
			a.activeDialog.Deny()
			a.tui.PopOverlay()
			a.activeDialog = nil
		}
		a.tui.RequestRender()
		return
	}

	// 2. Ctrl+C: if agent running, abort; else exit
	if k.Type == key.KeyCtrlC {
		if a.agentRunning.Load() && a.activeAgent != nil {
			a.activeAgent.Abort()
		} else {
			a.cancelFn()
		}
		return
	}

	// 3. Ctrl+D: always exit
	if k.Type == key.KeyCtrlD {
		a.cancelFn()
		return
	}

	// 4. BackTab (Shift+Tab): toggle Plan/Edit mode
	if k.Type == key.KeyBackTab {
		a.ToggleMode()
		a.updateFooter()
		a.tui.RequestRender()
		return
	}

	// 5. Enter: submit prompt if agent not running and editor has text
	if k.Type == key.KeyEnter && !a.agentRunning.Load() {
		text := a.editor.Text()
		if text != "" {
			a.submitPrompt(text)
			return
		}
	}

	// 6. Default: route to editor
	a.editor.HandleKey(k)
	a.tui.RequestRender()
}

// submitPrompt sends user input to the agent or dispatches a slash command.
func (a *App) submitPrompt(text string) {
	container := a.tui.Container()

	// Clear editor
	a.editor.SetText("")

	// Check for slash commands before sending to agent
	if commands.IsCommand(text) {
		a.handleSlashCommand(container, text)
		return
	}

	// Remove editor+footer temporarily
	container.Remove(a.editor)
	container.Remove(a.footer)

	// Add user message
	container.Add(components.NewUserMessage(text))

	// Add assistant message placeholder
	a.current = components.NewAssistantMessage()
	container.Add(a.current)

	// Re-add editor+footer at bottom
	container.Add(a.editor)
	container.Add(a.footer)

	// Append to conversation history
	a.messages = append(a.messages, ai.NewTextMessage(ai.RoleUser, text))

	a.tui.RequestRender()

	// Run agent in background
	go a.runAgent()
}

// handleSlashCommand dispatches a slash command and displays the result.
func (a *App) handleSlashCommand(container *tuipkg.Container, text string) {
	cwd, _ := os.Getwd()
	modelName := "none"
	if a.model != nil {
		modelName = a.model.Name
	}

	cmdCtx := &commands.CommandContext{
		Model:    modelName,
		Mode:     a.mode.String(),
		Version:  a.version,
		CWD:      cwd,
		Messages: len(a.messages),
		ClearHistory: func() {
			a.messages = nil
		},
		ToggleMode: func() {
			a.ToggleMode()
		},
		GetMode: func() string {
			return a.mode.String()
		},
		ExitFn: func() {
			a.cancelFn()
		},
	}

	result, err := a.cmdRegistry.Dispatch(cmdCtx, text)

	// Display result
	container.Remove(a.editor)
	container.Remove(a.footer)

	container.Add(components.NewUserMessage(text))

	msg := components.NewAssistantMessage()
	if err != nil {
		msg.AppendText(fmt.Sprintf("Error: %v", err))
	} else {
		msg.AppendText(result)
	}
	container.Add(msg)

	container.Add(a.editor)
	container.Add(a.footer)

	a.updateFooter()
	a.tui.RequestRender()
}

// runAgent executes the agent loop, streaming events to TUI components.
func (a *App) runAgent() {
	a.agentRunning.Store(true)
	defer a.agentRunning.Store(false)

	if a.provider == nil || a.model == nil {
		if a.current != nil {
			a.current.AppendText("Error: no provider or model configured")
			a.tui.RequestRender()
		}
		return
	}

	// Build tool definitions for LLM context
	aiTools := make([]ai.Tool, 0, len(a.tools))
	for _, t := range a.tools {
		schema := t.Parameters
		if schema == nil {
			schema = json.RawMessage(`{}`)
		}
		aiTools = append(aiTools, ai.Tool{
			Name:        t.Name,
			Description: t.Description,
			Parameters:  schema,
		})
	}

	llmCtx := &ai.Context{
		System:   a.systemPrompt,
		Messages: a.messages,
		Tools:    aiTools,
	}

	opts := &ai.StreamOptions{
		MaxTokens: 16384,
	}
	if a.model.MaxOutputTokens > 0 {
		opts.MaxTokens = a.model.MaxOutputTokens
	}

	// Create agent with permission checking
	permCheckFn := func(tool string, args map[string]any) error {
		return a.checker.Check(tool, args)
	}
	ag := agent.NewWithPermissions(a.provider, a.model, a.tools, permCheckFn)
	a.activeAgent = ag

	// Track response timing for tok/s calculation
	a.lastResponseStart.Store(time.Now().UnixNano())
	a.lastOutputTokens.Store(0)
	a.tokPerSec.Store(0)

	events := ag.Prompt(a.ctx, llmCtx, opts)

	container := a.tui.Container()
	toolExecs := make(map[string]*components.ToolExec)

	for evt := range events {
		switch evt.Type {
		case agent.EventAssistantText:
			if a.current != nil {
				a.current.AppendText(evt.Text)
			}

		case agent.EventAssistantThinking:
			if a.current != nil {
				a.current.SetThinking(evt.Text)
			}

		case agent.EventToolStart:
			argsStr := ""
			if evt.ToolArgs != nil {
				if data, err := json.Marshal(evt.ToolArgs); err == nil {
					argsStr = string(data)
				}
			}
			te := components.NewToolExec(evt.ToolName, argsStr)
			toolExecs[evt.ToolID] = te
			// Insert before editor
			container.Remove(a.editor)
			container.Remove(a.footer)
			container.Add(te)
			container.Add(a.editor)
			container.Add(a.footer)

		case agent.EventToolUpdate:
			if te, ok := toolExecs[evt.ToolID]; ok {
				te.AppendOutput(evt.Text)
			}

		case agent.EventToolEnd:
			if te, ok := toolExecs[evt.ToolID]; ok {
				errMsg := ""
				if evt.ToolResult != nil && evt.ToolResult.IsError {
					errMsg = evt.ToolResult.Content
				}
				te.SetDone(errMsg)
			}

		case agent.EventUsageUpdate:
			if evt.Usage != nil {
				a.totalInputTokens.Add(int64(evt.Usage.InputTokens))
				a.totalOutputTokens.Add(int64(evt.Usage.OutputTokens))
				a.lastOutputTokens.Add(int64(evt.Usage.OutputTokens))
				startNano := a.lastResponseStart.Load()
				if startNano > 0 {
					elapsed := time.Since(time.Unix(0, startNano)).Seconds()
					if elapsed > 0 {
						tps := float64(a.lastOutputTokens.Load()) / elapsed
						a.tokPerSec.Store(int64(tps * 10))
					}
				}
				a.updateFooter()
			}

		case agent.EventError:
			if a.current != nil && evt.Error != nil {
				a.current.AppendText(fmt.Sprintf("\n[error: %v]", evt.Error))
			}
		}

		a.tui.RequestRender()
	}

	// Collect assistant response for conversation history
	if a.current != nil {
		// Build assistant message from accumulated text
		var assistantContent []ai.Content
		// The agent already appended messages to llmCtx.Messages inside its loop;
		// we sync our local messages slice with the final state.
		a.messages = llmCtx.Messages
	_ = assistantContent // history is tracked via llmCtx
	}

	// Clean up completed tool components to prevent unbounded container growth.
	// The user has already seen the streaming output; completed tools render as
	// one-line status indicators, but accumulating hundreds still degrades render.
	container = a.tui.Container()
	for _, te := range toolExecs {
		container.Remove(te)
	}

	a.activeAgent = nil
	a.updateFooter()
	a.tui.RequestRender()
}

// askPermission shows a permission dialog overlay and blocks until the user responds.
func (a *App) askPermission(tool string, args map[string]any) (bool, error) {
	argsStr := ""
	if args != nil {
		if data, err := json.Marshal(args); err == nil {
			argsStr = string(data)
		}
	}

	dialog := components.NewPermissionDialog(tool, argsStr)
	a.activeDialog = dialog

	a.tui.PushOverlay(tuipkg.Overlay{
		Component: dialog,
		Position:  tuipkg.OverlayCenter,
	})
	a.tui.RequestRender()

	// Blocks until user responds (y/a/n/Esc in onKey)
	resp := dialog.Wait()

	switch resp {
	case components.PermAllowAlways:
		// Persist an allow rule so future invocations of this tool are auto-allowed
		specifier := permission.ExtractSpecifier(tool, args)
		if specifier != "" {
			a.checker.AddGlobAllowRule(tool, specifier)
		} else {
			a.checker.AddAllowRule(permission.Rule{Tool: tool})
		}
		return true, nil
	case components.PermAllow:
		return true, nil
	default:
		return false, nil
	}
}

// updateFooter refreshes the footer content with cwd, git branch, mode, model, and token stats.
func (a *App) updateFooter() {
	if a.footer == nil {
		return
	}

	var parts []string

	// CWD (shortened to ~ if under HOME)
	cwd, _ := os.Getwd()
	if home, err := os.UserHomeDir(); err == nil && strings.HasPrefix(cwd, home) {
		cwd = "~" + cwd[len(home):]
	}
	cwdPart := cwd
	if a.gitBranch != "" {
		cwdPart = fmt.Sprintf("%s (%s)", filepath.Base(cwd), a.gitBranch)
	}
	parts = append(parts, cwdPart)

	// Mode + hint
	parts = append(parts, a.ModeLabel())

	// Model
	modelName := "none"
	if a.model != nil {
		modelName = a.model.Name
	}
	parts = append(parts, modelName)

	// Token stats (only if we have any)
	inTok := int(a.totalInputTokens.Load())
	outTok := int(a.totalOutputTokens.Load())
	if inTok > 0 || outTok > 0 {
		stats := fmt.Sprintf("\u2191%s \u2193%s",
			formatTokenCount(inTok),
			formatTokenCount(outTok))
		tps := float64(a.tokPerSec.Load()) / 10.0
		if tps > 0 {
			stats = fmt.Sprintf("\u2191%s \u2193%s %.1f tok/s",
				formatTokenCount(inTok),
				formatTokenCount(outTok),
				tps)
		}
		parts = append(parts, stats)
	}

	a.footer.SetContent(strings.Join(parts, " | "))
}

// formatTokenCount formats a token count for compact display.
func formatTokenCount(n int) string {
	switch {
	case n >= 1_000_000:
		return fmt.Sprintf("%.1fM", float64(n)/1_000_000)
	case n >= 1_000:
		return fmt.Sprintf("%.0fk", float64(n)/1_000)
	default:
		return fmt.Sprintf("%d", n)
	}
}

// detectGitBranch returns the current git branch name, or empty string.
func detectGitBranch() string {
	out, err := exec.Command("git", "rev-parse", "--abbrev-ref", "HEAD").Output()
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(out))
}

// ToggleMode switches between Plan and Edit modes.
// Saves the current permission mode when entering Plan;
// restores it when switching back to Edit.
func (a *App) ToggleMode() {
	switch a.mode {
	case ModePlan:
		a.mode = ModeEdit
		a.checker.SetMode(a.editPermMode)
	case ModeEdit:
		a.mode = ModePlan
		a.editPermMode = a.checker.Mode()
		a.checker.SetMode(permission.ModePlan)
	}
}

// Mode returns the current mode.
func (a *App) Mode() Mode {
	return a.mode
}

// ModeLabel returns the display label for the footer.
// Shows sub-mode for non-normal edit modes (e.g., "[EDIT:accept-edits]").
func (a *App) ModeLabel() string {
	switch a.mode {
	case ModePlan:
		return "[PLAN] Shift+Tab -> Edit"
	case ModeEdit:
		permMode := a.checker.Mode()
		if permMode != permission.ModeNormal && permMode != permission.ModePlan {
			return fmt.Sprintf("[EDIT:%s] Shift+Tab -> Plan", permMode)
		}
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
	a.editPermMode = permission.ModeYolo
	a.checker.SetMode(permission.ModeYolo)
}

// SetAcceptEditsMode enables accept-edits mode (auto-allow edits, prompt for bash).
func (a *App) SetAcceptEditsMode() {
	a.mode = ModeEdit
	a.editPermMode = permission.ModeAcceptEdits
	a.checker.SetMode(permission.ModeAcceptEdits)
}

// StatusLine returns the status information for the footer.
func (a *App) StatusLine(model string, totalCost float64) string {
	return fmt.Sprintf("%s | %s | $%.4f", a.ModeLabel(), model, totalCost)
}
