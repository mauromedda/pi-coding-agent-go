// ABOUTME: Interactive TUI mode: main loop, mode switching (Plan/Edit), agent wiring
// ABOUTME: Orchestrates agent session, TUI rendering, keyboard input, and permission dialogs

package interactive

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"sync/atomic"
	"time"

	"github.com/mauromedda/pi-coding-agent-go/pkg/tui/clipboard"

	"github.com/mauromedda/pi-coding-agent-go/internal/agent"
	"github.com/mauromedda/pi-coding-agent-go/internal/commands"
	"github.com/mauromedda/pi-coding-agent-go/internal/config"
	"github.com/mauromedda/pi-coding-agent-go/internal/mode/interactive/components"
	"github.com/mauromedda/pi-coding-agent-go/internal/permission"
	"github.com/mauromedda/pi-coding-agent-go/internal/session"
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
	Terminal             terminal.Terminal
	Provider             ai.ApiProvider
	Model                *ai.Model
	Tools                []*agent.AgentTool
	Checker              *permission.Checker
	SystemPrompt         string
	Version              string
	StatusEngine         *statusline.Engine
	AutoCompactThreshold int
	Hooks                map[string][]config.HookDef
	ScopedModels         *config.ScopedModelsConfig
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
	editor       *component.Editor
	editorSep    *components.Separator // permanent separator above editor
	editorSepBot *components.Separator // permanent separator below editor
	footer       *components.Footer
	current      *components.AssistantMessage // current streaming response

	// File mention selector (for @ file autocomplete)
	fileMentionSelector *component.FileMentionSelector
	fileMentionVisible  bool

	// Interactive panel components
	sessionTree    *components.SessionTree
	sessionTreeVis bool
	hookManager    *components.HookManager
	hookManagerVis bool
	permManager    *components.PermissionManager
	permManagerVis bool

	// Config-provided data
	hookDefs     map[string][]config.HookDef
	scopedModels *config.ScopedModelsConfig

	// State
	sessionID    string // persisted session ID (empty if not yet saved)
	agentRunning atomic.Bool
	activeAgent  *agent.Agent
	activeDialog atomic.Pointer[components.PermissionDialog]
	messages     []ai.Message // conversation history
	cmdRegistry  *commands.Registry
	msgQueue     *MessageQueue // follow-up message queue

	// Token stats + context info
	totalInputTokens  atomic.Int64
	totalOutputTokens atomic.Int64
	lastContextTokens atomic.Int64 // input tokens from latest LLM call (context size)
	lastResponseStart atomic.Int64 // UnixNano timestamp
	lastOutputTokens  atomic.Int64 // output tokens for current response
	tokPerSec         atomic.Int64 // tok/s × 10 (fixed-point)
	gitBranch         string

	// Current thinking level (cycled via alt+t)
	thinkingLevel config.ThinkingLevel

	// Saved permission mode for Plan/Edit toggle
	editPermMode permission.Mode

	// External status line engine (optional)
	statusEngine *statusline.Engine

	// Configurable auto-compact threshold (percentage 1-100; 0 = default 80%)
	autoCompactThreshold int
}

// NewFromDeps creates a fully-wired interactive app from dependencies.
func NewFromDeps(deps AppDeps) *App {
	ctx, cancel := context.WithCancel(context.Background())
	app := &App{
		tui:                  tuipkg.New(deps.Terminal, 80, 24),
		mode:                 ModePlan,
		checker:              deps.Checker,
		ctx:                  ctx,
		cancelFn:             cancel,
		term:                 deps.Terminal,
		provider:             deps.Provider,
		model:                deps.Model,
		tools:                deps.Tools,
		systemPrompt:         deps.SystemPrompt,
		version:              deps.Version,
		cmdRegistry:          commands.NewRegistry(),
		msgQueue:             NewMessageQueue(),
		statusEngine:         deps.StatusEngine,
		autoCompactThreshold: deps.AutoCompactThreshold,
		hookDefs:             deps.Hooks,
		scopedModels:         deps.ScopedModels,
	}
	// Initialize file mention selector
	app.initFileMentionSelector()
	return app
}

// New creates a new interactive app (backwards-compatible constructor).
func New(writer tuipkg.Writer, width, height int, checker *permission.Checker) *App {
	ctx, cancel := context.WithCancel(context.Background())
	app := &App{
		tui:      tuipkg.New(writer, width, height),
		mode:     ModePlan,
		checker:  checker,
		ctx:      ctx,
		cancelFn: cancel,
	}
	// Initialize file mention selector
	app.initFileMentionSelector()
	return app
}

// initFileMentionSelector initializes the file mention selector component.
func (a *App) initFileMentionSelector() {
	cwd, _ := os.Getwd()
	a.fileMentionSelector = component.NewFileMentionSelector(cwd, cwd)
	// Scan project files in background
	go func() {
		defer func() { recover() }() //nolint:errcheck // non-critical goroutine
		_ = a.fileMentionSelector.ScanProject()
	}()
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

	// Set up editor separators, editor, and footer
	a.editorSep = components.NewSeparator()
	a.editorSepBot = components.NewSeparator()
	a.editor = component.NewEditor()
	a.editor.SetFocused(true)
	a.editor.SetPrompt("❯ ")
	a.editor.SetPlaceholder("Try \"how does <filepath> work?\"")
	a.footer = components.NewFooter()
	a.updateFooter()

	container := a.tui.Container()
	container.Add(a.editorSep)
	container.Add(a.editor)
	container.Add(a.editorSepBot)
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
	go func() {
		defer terminal.RecoverGoroutine(a.term)
		stdinBuf.Start(a.ctx)
	}()

	// Block until context done
	<-a.ctx.Done()
	return nil
}

// printWelcome adds the welcome banner and separator to the container.
func (a *App) printWelcome(container *tuipkg.Container) {
	ver := a.version
	if ver == "" {
		ver = "dev"
	}
	modelName := "none"
	if a.model != nil {
		modelName = a.model.Name
	}
	cwd, _ := os.Getwd()
	if home, _ := os.UserHomeDir(); home != "" && strings.HasPrefix(cwd, home) {
		cwd = "~" + cwd[len(home):]
	}

	welcome := components.NewWelcomeMessage(ver, modelName, cwd, len(a.tools))

	container.Remove(a.editorSep)
	container.Remove(a.editor)
	container.Remove(a.editorSepBot)
	container.Remove(a.footer)
	container.Add(welcome)
	container.Add(a.editorSep)
	container.Add(a.editor)
	container.Add(a.editorSepBot)
	container.Add(a.footer)
}

// onKey handles a single key event from the StdinBuffer goroutine.
func (a *App) onKey(k key.Key) {
	// 1. Active overlay (permission dialog): route y/a/n/Esc
	if dialog := a.activeDialog.Load(); dialog != nil {
		switch k.Type {
		case key.KeyRune:
			switch k.Rune {
			case 'y', 'Y':
				dialog.Allow()
				a.tui.PopOverlay()
				a.activeDialog.Store(nil)
			case 'a', 'A':
				dialog.AllowAlways()
				a.tui.PopOverlay()
				a.activeDialog.Store(nil)
			case 'n', 'N':
				dialog.Deny()
				a.tui.PopOverlay()
				a.activeDialog.Store(nil)
			}
		case key.KeyEscape:
			dialog.Deny()
			a.tui.PopOverlay()
			a.activeDialog.Store(nil)
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

	// 3b. Ctrl+G: open external editor
	if k.Type == key.KeyCtrlG && !a.agentRunning.Load() {
		a.openExternalEditor()
		return
	}

	// 3c. Ctrl+O: toggle tool call expand/collapse (if agent not running)
	if k.Type == key.KeyCtrlO && !a.agentRunning.Load() && a.current != nil {
		tcList := a.current.GetToolCalls()
		if len(tcList) > 0 {
			// Toggle expand on the last tool call
			a.current.ToggleToolCallExpand(len(tcList) - 1)
			a.tui.RequestRender()
		}
		return
	}

	// 4. BackTab (Shift+Tab): toggle Plan/Edit mode
	if k.Type == key.KeyBackTab {
		a.ToggleMode()
		a.updateFooter()
		a.tui.RequestRender()
		return
	}

	// 4b. Shift+Ctrl+P: cycle through scoped models
	if k.Type == key.KeyRune && k.Ctrl && k.Shift && (k.Rune == 'p' || k.Rune == 'P') {
		a.cycleModelForward()
		return
	}

	// 4c. Alt+T: cycle thinking level
	if k.Type == key.KeyRune && k.Alt && (k.Rune == 't' || k.Rune == 'T') {
		nextIdx := (a.thinkingLevel.Index() + 1) % 6
		a.thinkingLevel = config.ThinkingLevelFromIndex(nextIdx)
		a.footer.SetThinkingLevel(a.thinkingLevel)
		a.tui.RequestRender()
		return
	}

	// 5a. Alt+Enter: queue follow-up message while agent runs
	if k.Type == key.KeyEnter && k.Alt {
		text := a.editor.Text()
		if text != "" {
			a.msgQueue.Push(text)
			a.editor.SetText("")
			a.footer.SetQueuedCount(a.msgQueue.Count())
			a.tui.RequestRender()
		}
		return
	}

	// 5b. Alt+Up/Down: cycle through message history when agent not running
	if k.Alt && !a.agentRunning.Load() {
		switch k.Type {
		case key.KeyUp:
			if msg := a.msgQueue.Prev(); msg != "" {
				a.editor.SetText(msg)
				a.tui.RequestRender()
			}
			return
		case key.KeyDown:
			if msg := a.msgQueue.Next(); msg != "" {
				a.editor.SetText(msg)
				a.tui.RequestRender()
			}
			return
		}
	}

	// 5c. Alt+Left: start editing current queued message
	if k.Type == key.KeyLeft && k.Alt && !a.agentRunning.Load() {
		if msg := a.msgQueue.StartEdit(); msg != "" {
			a.editor.SetText(msg)
			a.tui.RequestRender()
		}
		return
	}

	// 5d. Alt+Right: commit edit and move to next queued message
	if k.Type == key.KeyRight && k.Alt && !a.agentRunning.Load() {
		if a.msgQueue.EditMode() {
			a.msgQueue.EditMessage(a.editor.Text())
			a.msgQueue.CommitEdit()
			// Move to next message
			if msg := a.msgQueue.Next(); msg != "" {
				a.editor.SetText(msg)
			}
			a.tui.RequestRender()
		}
		return
	}

	// 5c. Enter: submit prompt if agent not running and editor has text
	if k.Type == key.KeyEnter && !a.agentRunning.Load() {
		text := a.editor.Text()
		if text != "" {
			a.submitPrompt(text)
			return
		}
	}

	// 7. @ key: show file mention selector (if agent not running)
	if k.Type == key.KeyRune && k.Rune == '@' && !a.agentRunning.Load() && !a.fileMentionVisible {
		a.showFileMentionSelector()
		a.tui.RequestRender()
		return
	}

	// 8. Handle file mention selector input if visible
	if a.fileMentionVisible {
		if handled := a.handleFileMentionInput(k); handled {
			return
		}
	}

	// 8a. Handle interactive panel input if visible
	if a.sessionTreeVis {
		if handled := a.handleSessionTreeInput(k); handled {
			return
		}
	}
	if a.hookManagerVis {
		if handled := a.handleHookManagerInput(k); handled {
			return
		}
	}
	if a.permManagerVis {
		if handled := a.handlePermManagerInput(k); handled {
			return
		}
	}

	// 9. Default: route to editor
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
	container.Remove(a.editorSep)
	container.Remove(a.editor)
	container.Remove(a.editorSepBot)
	container.Remove(a.footer)

	// Add user message
	container.Add(components.NewUserMessage(text))

	// Re-add separator+editor+footer at bottom
	container.Add(a.editorSep)
	container.Add(a.editor)
	container.Add(a.editorSepBot)
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
		ClearTUI: func() {
			container.Clear()
			// Re-add permanent components
			container.Add(a.editorSep)
			container.Add(a.editor)
			container.Add(a.editorSepBot)
			container.Add(a.footer)
			a.current = nil
			// Reset token stats
			a.totalInputTokens.Store(0)
			a.totalOutputTokens.Store(0)
			a.lastContextTokens.Store(0)
			a.tokPerSec.Store(0)
			a.tui.FullClear()
			a.updateFooter()
		},
		CompactFn: func() string {
			prev := len(a.messages)
			compacted, summary, err := session.Compact(a.messages)
			if err != nil {
				return fmt.Sprintf("Error: %v", err)
			}
			if len(compacted) == prev {
				return "Nothing to compact."
			}
			a.messages = compacted
			return fmt.Sprintf("Compacted %d → %d messages.\n%s", prev, len(a.messages), summary)
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
		ReloadFn: func() (string, error) {
			return "Configuration reloaded.", nil
		},
		SessionTreeFn: func() string {
			a.showSessionTree()
			return "Session tree opened."
		},
		HookManagerFn: func() string {
			a.showHookManager()
			return "Hook manager opened."
		},
		PermissionManagerFn: func() string {
			a.showPermissionManager()
			return "Permission manager opened."
		},
		ScopedModelsFn: func() string {
			if a.scopedModels == nil {
				return "No scoped models configured."
			}
			var b strings.Builder
			b.WriteString("Scoped models:\n")
			current := ""
			if a.model != nil {
				current = a.model.Name
			}
			for _, m := range a.scopedModels.Models {
				marker := "  "
				if m.Name == current {
					marker = "* "
				}
				line := fmt.Sprintf("%s%s", marker, m.Name)
				if m.Thinking != config.ThinkingOff {
					line += fmt.Sprintf(" [thinking: %s]", m.Thinking.String())
				}
				b.WriteString(line + "\n")
			}
			return b.String()
		},
		CopyLastMessageFn: func() (string, error) {
			return a.copyLastAssistantMessage()
		},
		NewSessionFn: func() {
			a.messages = nil
			a.current = nil
			a.totalInputTokens.Store(0)
			a.totalOutputTokens.Store(0)
			a.lastContextTokens.Store(0)
			a.tokPerSec.Store(0)
			container.Clear()
			a.printWelcome(container)
			a.updateFooter()
			a.tui.FullClear()
			a.tui.RequestRender()
		},
		ForkSessionFn: func() (string, error) {
			if a.sessionID == "" {
				return "", fmt.Errorf("no active session to fork (session not persisted)")
			}
			sessDir := config.SessionsDir()
			result, err := session.Fork(sessDir, a.sessionID)
			if err != nil {
				return "", err
			}
			return result.NewID, nil
		},
		ListSessionsFn: func() string {
			sessions, err := session.ListSessions()
			if err != nil {
				return fmt.Sprintf("Error listing sessions: %v", err)
			}
			if len(sessions) == 0 {
				return "No sessions found."
			}
			var b strings.Builder
			b.WriteString("Sessions:\n")
			for i, s := range sessions {
				if i >= 20 {
					fmt.Fprintf(&b, "  ... and %d more\n", len(sessions)-20)
					break
				}
				fmt.Fprintf(&b, "  %s  %s  %s\n", s.ID[:8], s.Model, s.CWD)
			}
			return b.String()
		},
	}

	result, err := a.cmdRegistry.Dispatch(cmdCtx, text)

	// /clear already reset the TUI via ClearTUI callback; skip normal display
	if strings.TrimSpace(text) == "/clear" {
		a.tui.RequestRender()
		return
	}

	// Display result
	container.Remove(a.editorSep)
	container.Remove(a.editor)
	container.Remove(a.editorSepBot)
	container.Remove(a.footer)

	container.Add(components.NewUserMessage(text))

	msg := components.NewAssistantMessage()
	if err != nil {
		msg.AppendText(fmt.Sprintf("Error: %v", err))
	} else {
		msg.AppendText(result)
	}
	container.Add(msg)

	container.Add(a.editorSep)
	container.Add(a.editor)
	container.Add(a.editorSepBot)
	container.Add(a.footer)

	a.updateFooter()
	a.tui.RequestRender()
}

// showFileMentionSelector displays the file mention selector component.
func (a *App) showFileMentionSelector() {
	a.fileMentionVisible = true

	// Add file mention selector to container
	container := a.tui.Container()
	container.Add(a.fileMentionSelector)
}

// handleFileMentionInput handles input for the file mention selector.
func (a *App) handleFileMentionInput(k key.Key) bool {
	if !a.fileMentionVisible || a.fileMentionSelector == nil {
		return false
	}

	switch k.Type {
	case key.KeyEscape:
		// Cancel file mention selection
		a.hideFileMentionSelector()
		a.tui.RequestRender()
		return true

	case key.KeyEnter:
		// Accept selection and insert into editor
		filePath := a.fileMentionSelector.SelectionAccepted()
		a.hideFileMentionSelector()

		// Insert @filename into editor
		if filePath != "" {
			currentText := a.editor.Text()
			newText := currentText + "@" + filePath
			a.editor.SetText(newText)
		}
		a.tui.RequestRender()
		return true

	case key.KeyUp, key.KeyDown:
		// Navigation handled by selector
		a.fileMentionSelector.HandleInput(k.String())
		a.tui.RequestRender()
		return true

	case key.KeyTab:
		// Accept selection (same as Enter)
		filePath := a.fileMentionSelector.SelectionAccepted()
		a.hideFileMentionSelector()
		if filePath != "" {
			currentText := a.editor.Text()
			newText := currentText + "@" + filePath
			a.editor.SetText(newText)
		}
		a.tui.RequestRender()
		return true

	case key.KeyBackspace:
		// Trim the filter; if already empty, close the selector
		filter := a.fileMentionSelector.Filter()
		if len(filter) > 0 {
			a.fileMentionSelector.SetFilter(filter[:len(filter)-1])
		} else {
			a.hideFileMentionSelector()
		}
		a.tui.RequestRender()
		return true

	case key.KeyRune:
		// Filter by typing
		a.fileMentionSelector.SetFilter(a.fileMentionSelector.Filter() + string(k.Rune))
		a.tui.RequestRender()
		return true
	}

	return false
}

// hideFileMentionSelector hides the file mention selector component.
func (a *App) hideFileMentionSelector() {
	a.fileMentionVisible = false

	// Remove selector from container
	container := a.tui.Container()
	container.Remove(a.fileMentionSelector)
}

// newAssistantSegment creates a new AssistantMessage and inserts it
// into the container just before the editor/footer group.
func (a *App) newAssistantSegment() *components.AssistantMessage {
	msg := components.NewAssistantMessage()
	container := a.tui.Container()
	container.Remove(a.editorSep)
	container.Remove(a.editor)
	container.Remove(a.editorSepBot)
	container.Remove(a.footer)
	container.Add(msg)
	container.Add(a.editorSep)
	container.Add(a.editor)
	container.Add(a.editorSepBot)
	container.Add(a.footer)
	return msg
}

// runAgent executes the agent loop, streaming events to TUI components.
func (a *App) runAgent() {
	defer terminal.RecoverGoroutine(a.term)
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

	toolCalls := make(map[string]*components.ToolCall)

	for evt := range events {
		switch evt.Type {
		case agent.EventAssistantText:
			if a.current == nil {
				a.current = a.newAssistantSegment()
			}
			a.current.AppendText(evt.Text)

		case agent.EventAssistantThinking:
			if a.current == nil {
				a.current = a.newAssistantSegment()
			}
			a.current.SetThinking(evt.Text)

		case agent.EventToolStart:
			argsStr := ""
			if evt.ToolArgs != nil {
				if data, err := json.Marshal(evt.ToolArgs); err == nil {
					argsStr = string(data)
				}
			}
			tc := components.NewToolCall(evt.ToolName, argsStr)
			toolCalls[evt.ToolID] = tc

		case agent.EventToolUpdate:
			if tc, ok := toolCalls[evt.ToolID]; ok {
				tc.SetDone(evt.Text, "")
			}

		case agent.EventToolEnd:
			if tc, ok := toolCalls[evt.ToolID]; ok {
				errMsg := ""
				output := evt.Text
				if evt.ToolResult != nil && evt.ToolResult.IsError {
					errMsg = evt.ToolResult.Content
				}
				tc.SetDone(output, errMsg)
				if a.current != nil {
					a.current.AddToolCall(tc)
				}
			}

		case agent.EventUsageUpdate:
			if evt.Usage != nil {
				a.totalInputTokens.Add(int64(evt.Usage.InputTokens))
				a.totalOutputTokens.Add(int64(evt.Usage.OutputTokens))
				a.lastContextTokens.Store(int64(evt.Usage.InputTokens))
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
			if evt.Error != nil {
				if a.current == nil {
					a.current = a.newAssistantSegment()
				}
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

	a.activeAgent = nil
	a.autoCompactIfNeeded()
	a.footer.SetQueuedCount(a.msgQueue.Count())
	a.updateFooter()
	a.tui.RequestRender()

	// Auto-submit queued follow-up messages
	if a.msgQueue.HasMessages() {
		msg := a.msgQueue.Pop()
		a.footer.SetQueuedCount(a.msgQueue.Count())
		a.submitPrompt(msg)
	}
}

// compactionThreshold returns the effective auto-compact threshold percentage.
// Falls back to 80% if the configured value is out of range.
func (a *App) compactionThreshold() int {
	if a.autoCompactThreshold > 0 && a.autoCompactThreshold <= 100 {
		return a.autoCompactThreshold
	}
	return 80
}

// autoCompactIfNeeded compacts messages if context occupation >= threshold.
func (a *App) autoCompactIfNeeded() {
	if a.model == nil || a.model.MaxTokens == 0 {
		return
	}
	pct := int(a.lastContextTokens.Load()) * 100 / a.model.MaxTokens
	if pct < a.compactionThreshold() {
		return
	}
	compacted, _, err := session.Compact(a.messages)
	if err != nil {
		return
	}
	if len(compacted) < len(a.messages) {
		a.messages = compacted
	}
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
	a.activeDialog.Store(dialog)

	a.tui.PushOverlay(tuipkg.Overlay{
		Component: dialog,
		Position:  tuipkg.OverlayCenter,
	})
	a.tui.RequestRender()

	// Blocks until user responds (y/a/n/Esc in onKey) or context is cancelled.
	resp := dialog.WaitContext(a.ctx)

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

// updateFooter refreshes the rich two-line footer with all session info.
func (a *App) updateFooter() {
	if a.footer == nil {
		return
	}

	// Line 1: CWD (shortened to ~)
	cwd, _ := os.Getwd()
	if home, err := os.UserHomeDir(); err == nil && strings.HasPrefix(cwd, home) {
		cwd = "~" + cwd[len(home):]
	}
	a.footer.SetLine1(cwd)

	// Git branch (displayed with icon in footer)
	a.footer.SetGitBranch(a.gitBranch)

	// Model name
	modelName := "none"
	if a.model != nil {
		modelName = a.model.Name
	}
	a.footer.SetModel(modelName)

	// Permission mode
	if a.checker != nil {
		a.footer.SetPermissionMode(a.checker.Mode().String())
	}

	// Line 2: token stats (left) + model (right)
	inTok := int(a.totalInputTokens.Load())
	outTok := int(a.totalOutputTokens.Load())
	stats := fmt.Sprintf("↑%s ↓%s",
		formatTokenCount(inTok),
		formatTokenCount(outTok))
	tps := float64(a.tokPerSec.Load()) / 10.0
	if tps > 0 {
		stats = fmt.Sprintf("↑%s ↓%s %.1f tok/s",
			formatTokenCount(inTok),
			formatTokenCount(outTok),
			tps)
	}

	// Context occupation percentage
	ctxTokens := int(a.lastContextTokens.Load())
	if a.model != nil && a.model.MaxTokens > 0 && ctxTokens > 0 {
		pct := ctxTokens * 100 / a.model.MaxTokens
		a.footer.SetContextPct(pct)
	} else {
		a.footer.SetContextPct(0)
	}

	a.footer.SetLine2(stats, "")
	a.footer.SetModeLabel(a.ModeLabel())
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

// copyLastAssistantMessage finds the last assistant message and copies its text to clipboard.
func (a *App) copyLastAssistantMessage() (string, error) {
	for i := len(a.messages) - 1; i >= 0; i-- {
		if a.messages[i].Role == ai.RoleAssistant {
			var text strings.Builder
			for _, c := range a.messages[i].Content {
				if c.Type == ai.ContentText && c.Text != "" {
					if text.Len() > 0 {
						text.WriteString("\n")
					}
					text.WriteString(c.Text)
				}
			}
			if text.Len() == 0 {
				return "No text in last assistant message.", nil
			}
			if err := clipboard.Write(text.String()); err != nil {
				return "", fmt.Errorf("clipboard write: %w", err)
			}
			return "Copied to clipboard.", nil
		}
	}
	return "No assistant messages to copy.", nil
}

// cycleModelForward advances to the next scoped model (wrap-around).
func (a *App) cycleModelForward() {
	if a.scopedModels == nil || a.model == nil {
		return
	}
	next := a.scopedModels.CycleModels(a.model.Name, 1)
	if next == a.model.Name {
		return
	}
	a.model.Name = next
	a.footer.SetModel(next)
	a.tui.RequestRender()
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

// openExternalEditor opens $VISUAL/$EDITOR/vi with the current editor text.
func (a *App) openExternalEditor() {
	// Resolve editor command
	editor := os.Getenv("VISUAL")
	if editor == "" {
		editor = os.Getenv("EDITOR")
	}
	if editor == "" {
		editor = "vi"
	}

	// Write current text to a temp file
	tmpFile, err := os.CreateTemp("", ".pi-*.md")
	if err != nil {
		return
	}
	tmpPath := tmpFile.Name()
	defer os.Remove(tmpPath)

	_, _ = tmpFile.WriteString(a.editor.Text())
	_ = tmpFile.Close()

	// Exit raw mode so the editor can control the terminal
	_ = a.term.ExitRawMode()

	// Launch editor
	cmd := exec.Command(editor, tmpPath)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	_ = cmd.Run()

	// Read back the edited content
	data, err := os.ReadFile(tmpPath)
	if err == nil {
		a.editor.SetText(strings.TrimRight(string(data), "\n"))
	}

	// Re-enter raw mode and refresh TUI
	_ = a.term.EnterRawMode()
	a.tui.FullClear()
	a.tui.RequestRender()
}

// --- SessionTree panel ---

func (a *App) showSessionTree() {
	sessDir := config.SessionsDir()
	roots, err := components.LoadSessionsFromDir(sessDir)
	if err != nil {
		roots = nil
	}
	a.sessionTree = components.NewSessionTree(roots)
	a.sessionTreeVis = true

	container := a.tui.Container()
	container.Remove(a.editorSep)
	container.Remove(a.editor)
	container.Remove(a.editorSepBot)
	container.Remove(a.footer)
	container.Add(a.sessionTree)
	container.Add(a.editorSep)
	container.Add(a.editor)
	container.Add(a.editorSepBot)
	container.Add(a.footer)
	a.tui.RequestRender()
}

func (a *App) hideSessionTree() {
	if a.sessionTree != nil {
		a.tui.Container().Remove(a.sessionTree)
	}
	a.sessionTreeVis = false
	a.sessionTree = nil
	a.tui.RequestRender()
}

func (a *App) handleSessionTreeInput(k key.Key) bool {
	switch k.Type {
	case key.KeyEscape:
		a.hideSessionTree()
		return true
	case key.KeyEnter:
		if node := a.sessionTree.SelectedNode(); node != nil {
			a.hideSessionTree()
			// Resume the selected session (best effort)
			_ = a.resumeSessionByID(node.ID)
		}
		return true
	case key.KeyUp, key.KeyDown:
		a.sessionTree.HandleKey(k)
		a.tui.RequestRender()
		return true
	case key.KeyRune:
		// Filter by typing
		a.sessionTree.SetFilter(a.sessionTree.Filter() + string(k.Rune))
		a.tui.RequestRender()
		return true
	case key.KeyBackspace:
		f := a.sessionTree.Filter()
		if len(f) > 0 {
			a.sessionTree.SetFilter(f[:len(f)-1])
		}
		a.tui.RequestRender()
		return true
	}
	return false
}

func (a *App) resumeSessionByID(id string) error {
	// Verify the session exists by reading its records
	_, err := session.ReadRecords(id)
	return err
}

// --- HookManager panel ---

func (a *App) showHookManager() {
	hooks := components.ConvertFromConfig(a.hookDefs)
	a.hookManager = components.NewHookManager()
	a.hookManager.SetHooks(hooks)
	a.hookManagerVis = true

	container := a.tui.Container()
	container.Remove(a.editorSep)
	container.Remove(a.editor)
	container.Remove(a.editorSepBot)
	container.Remove(a.footer)
	container.Add(a.hookManager)
	container.Add(a.editorSep)
	container.Add(a.editor)
	container.Add(a.editorSepBot)
	container.Add(a.footer)
	a.tui.RequestRender()
}

func (a *App) hideHookManager() {
	if a.hookManager != nil {
		a.tui.Container().Remove(a.hookManager)
	}
	a.hookManagerVis = false
	a.hookManager = nil
	a.tui.RequestRender()
}

func (a *App) handleHookManagerInput(k key.Key) bool {
	switch k.Type {
	case key.KeyEscape:
		a.hideHookManager()
		return true
	case key.KeyEnter, key.KeyUp, key.KeyDown:
		a.hookManager.HandleKey(k)
		a.tui.RequestRender()
		return true
	}
	return false
}

// --- PermissionManager panel ---

func (a *App) showPermissionManager() {
	rules := a.checker.Rules()
	rulesPtrs := make([]*permission.Rule, len(rules))
	for i := range rules {
		rulesPtrs[i] = &rules[i]
	}
	a.permManager = components.NewPermissionManager()
	a.permManager.SetRules(rulesPtrs)
	a.permManagerVis = true

	container := a.tui.Container()
	container.Remove(a.editorSep)
	container.Remove(a.editor)
	container.Remove(a.editorSepBot)
	container.Remove(a.footer)
	container.Add(a.permManager)
	container.Add(a.editorSep)
	container.Add(a.editor)
	container.Add(a.editorSepBot)
	container.Add(a.footer)
	a.tui.RequestRender()
}

func (a *App) hidePermissionManager() {
	if a.permManager != nil {
		a.tui.Container().Remove(a.permManager)
	}
	a.permManagerVis = false
	a.permManager = nil
	a.tui.RequestRender()
}

func (a *App) handlePermManagerInput(k key.Key) bool {
	switch k.Type {
	case key.KeyEscape:
		a.hidePermissionManager()
		return true
	case key.KeyUp, key.KeyDown:
		a.permManager.HandleKey(k)
		a.tui.RequestRender()
		return true
	case key.KeyRune:
		// 'd' key deletes selected rule
		if k.Rune == 'd' {
			if rw := a.permManager.SelectedRule(); rw != nil {
				a.checker.RemoveRule(rw.Tool)
				a.permManager.RemoveRule()
				a.tui.RequestRender()
			}
			return true
		}
	}
	return false
}
