// ABOUTME: Root AppModel wiring all sub-models for the Bubble Tea interactive TUI
// ABOUTME: Handles message routing, overlay management, agent lifecycle, and key dispatch

package btea

import (
	"context"
	"encoding/json"
	"fmt"
	"os/exec"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/mauromedda/pi-coding-agent-go/internal/agent"
	"github.com/mauromedda/pi-coding-agent-go/internal/commands"
	"github.com/mauromedda/pi-coding-agent-go/internal/config"
	"github.com/mauromedda/pi-coding-agent-go/internal/permission"
	"github.com/mauromedda/pi-coding-agent-go/pkg/ai"
)

// Mode represents the current editing mode.
type Mode int

const (
	// ModePlan restricts the agent to read-only tools.
	ModePlan Mode = iota
	// ModeEdit allows the agent full tool access.
	ModeEdit
)

// String returns the human-readable label for the mode.
func (m Mode) String() string {
	switch m {
	case ModePlan:
		return "Plan"
	case ModeEdit:
		return "Edit"
	default:
		return "Unknown"
	}
}

// gitBranchMsg carries the detected git branch name.
type gitBranchMsg struct{ branch string }

// shared holds mutable state that must survive AppModel value copies.
// Bubble Tea copies the model on each Update; pointer fields are shared
// across copies. This avoids the need for a mutex: Bubble Tea's Update
// is single-threaded, and the goroutine only writes via Program.Send.
type shared struct {
	program     *tea.Program
	activeAgent *agent.Agent
	ctx         context.Context
	cancel      context.CancelFunc
}

// AppModel is the root Bubble Tea model for the interactive TUI.
type AppModel struct {
	sh *shared // survives value copies

	// State
	mode          Mode
	agentRunning  bool
	messages      []ai.Message
	width, height int

	// Sub-models (always present)
	editor EditorModel
	footer FooterModel

	// Content: ordered list of display models
	content []tea.Model // WelcomeModel, UserMsgModel, AssistantMsgModel, etc.

	// Overlay (nil = no overlay)
	overlay tea.Model

	// Dependencies
	deps AppDeps

	// Token stats
	totalInputTokens  int
	totalOutputTokens int

	// Session metadata
	gitBranch     string
	thinkingLevel config.ThinkingLevel

	// Command handling
	cmdRegistry *commands.Registry
}

// NewAppModel creates an AppModel wired with the given dependencies.
func NewAppModel(deps AppDeps) AppModel {
	ctx, cancel := context.WithCancel(context.Background())

	editor := NewEditorModel()
	editor = editor.SetFocused(true)
	editor = editor.SetPrompt("\u276f ")
	editor = editor.SetPlaceholder("Try \"how does <filepath> work?\"")

	modelName := ""
	if deps.Model != nil {
		modelName = deps.Model.Name
	}

	cwd := detectGitCWD()
	toolCount := len(deps.Tools)

	// Determine initial mode and permission label from PermissionMode.
	initialMode := ModeEdit
	permLabel := deps.PermissionMode.String()

	switch deps.PermissionMode {
	case permission.ModeYolo:
		initialMode = ModeEdit
	case permission.ModeAcceptEdits:
		initialMode = ModeEdit
	case permission.ModePlan:
		initialMode = ModePlan
	default:
		// ModeNormal, ModeDontAsk, etc.: default to Edit with their label.
		initialMode = ModeEdit
	}

	footer := NewFooterModel().
		WithPath(cwd).
		WithModel(modelName).
		WithModeLabel(initialMode.String()).
		WithPermissionMode(permLabel)

	welcome := NewWelcomeModel(deps.Version, modelName, cwd, toolCount)

	return AppModel{
		sh:          &shared{ctx: ctx, cancel: cancel},
		mode:        initialMode,
		editor:      editor,
		footer:      footer,
		content:     []tea.Model{welcome},
		deps:        deps,
		cmdRegistry: commands.NewRegistry(),
	}
}

// Init returns startup commands: detect git branch.
func (m AppModel) Init() tea.Cmd {
	return func() tea.Msg {
		return gitBranchMsg{branch: detectGitBranch()}
	}
}

// Update routes messages to the appropriate handler.
func (m AppModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	// --- Layout ---
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m = m.propagateSize(msg)
		return m, nil

	// --- Overlay lifecycle ---
	case DismissOverlayMsg:
		m.overlay = nil
		return m, nil

	// --- Overlay result messages (always handled by root, even when overlay is active) ---
	case CmdPaletteSelectMsg:
		m.overlay = nil
		m.editor = m.editor.SetText("/" + msg.Name)
		return m, nil

	case CmdPaletteDismissMsg:
		m.overlay = nil
		return m, nil

	case FileMentionSelectMsg:
		m.overlay = nil
		text := m.editor.Text()
		if text != "" && !strings.HasSuffix(text, " ") {
			text += " "
		}
		text += "@" + msg.RelPath
		m.editor = m.editor.SetText(text)
		return m, nil

	case FileMentionDismissMsg:
		m.overlay = nil
		return m, nil

	case ModelSelectedMsg:
		m.overlay = nil
		// Placeholder: model switch will be wired in a later phase
		return m, nil

	case ModelSelectorDismissMsg:
		m.overlay = nil
		return m, nil

	case SessionSelectedMsg:
		m.overlay = nil
		// Placeholder: session resume will be wired in a later phase
		return m, nil

	case SessionSelectorDismissMsg:
		m.overlay = nil
		return m, nil

	// --- Permission flow ---
	case PermissionRequestMsg:
		m.overlay = NewPermDialogModel(msg.Tool, msg.Args, msg.ReplyCh)
		return m, nil

	default:
		// Route to overlay if active (key presses, etc.)
		if m.overlay != nil {
			return m.updateOverlay(msg)
		}
	}

	// Non-overlay messages (only reached when no overlay is active)
	switch msg := msg.(type) {
	case gitBranchMsg:
		m.gitBranch = msg.branch
		m.footer = m.footer.WithGitBranch(msg.branch)
		return m, nil

	// --- Agent streaming events ---
	case AgentTextMsg:
		m = m.ensureAssistantMsg()
		m = m.updateLastAssistant(msg)
		return m, nil

	case AgentThinkingMsg:
		m = m.ensureAssistantMsg()
		m = m.updateLastAssistant(msg)
		return m, nil

	case AgentToolStartMsg:
		m = m.ensureAssistantMsg()
		m = m.updateLastAssistant(msg)
		return m, nil

	case AgentToolUpdateMsg:
		m = m.updateLastAssistant(msg)
		return m, nil

	case AgentToolEndMsg:
		m = m.updateLastAssistant(msg)
		return m, nil

	case AgentUsageMsg:
		if msg.Usage != nil {
			m.totalInputTokens += msg.Usage.InputTokens
			m.totalOutputTokens += msg.Usage.OutputTokens
		}
		updated, _ := m.footer.Update(msg)
		m.footer = updated.(FooterModel)
		return m, nil

	case AgentDoneMsg:
		m.agentRunning = false
		if len(msg.Messages) > 0 {
			m.messages = msg.Messages
		}
		return m, nil

	case AgentErrorMsg:
		m = m.ensureAssistantMsg()
		m = m.updateLastAssistant(AgentTextMsg{Text: fmt.Sprintf("\n[Error: %v]", msg.Err)})
		return m, nil

	// --- Key routing ---
	case tea.KeyMsg:
		return m.handleKey(msg)
	}

	return m, nil
}

// View renders the full TUI layout.
func (m AppModel) View() string {
	var sections []string

	for _, c := range m.content {
		sections = append(sections, c.View())
	}

	sep := NewSeparatorModel()
	sep.width = m.width

	sections = append(sections,
		sep.View(),
		m.editor.View(),
		sep.View(),
		m.footer.View(),
	)

	main := lipgloss.JoinVertical(lipgloss.Left, sections...)

	if m.overlay != nil {
		return main + "\n" + m.overlay.View()
	}

	return main
}

// --- Key handling ---

func (m AppModel) handleKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "ctrl+c":
		if m.agentRunning {
			m.abortAgent()
			return m, nil
		}
		return m, tea.Quit

	case "ctrl+d":
		return m, tea.Quit

	case "shift+tab":
		m = m.toggleMode()
		return m, nil

	case "alt+t":
		m = m.cycleThinking()
		return m, nil

	case "enter":
		if !m.agentRunning && !m.editor.IsEmpty() {
			text := m.editor.Text()
			return m.submitPrompt(text)
		}
		// Let editor handle enter for multi-line
		updated, cmd := m.editor.Update(msg)
		m.editor = updated.(EditorModel)
		return m, cmd

	default:
		// Check for "/" to open command palette
		if msg.Type == tea.KeyRunes && len(msg.Runes) == 1 {
			switch msg.Runes[0] {
			case '/':
				if !m.agentRunning && m.editor.IsEmpty() {
					m.overlay = m.buildCmdPalette()
					return m, nil
				}
			case '@':
				if !m.agentRunning {
					m.overlay = NewFileMentionModel("")
					return m, nil
				}
			}
		}
		// Route to editor
		updated, cmd := m.editor.Update(msg)
		m.editor = updated.(EditorModel)
		return m, cmd
	}
}

// --- Prompt submission ---

func (m AppModel) submitPrompt(text string) (AppModel, tea.Cmd) {
	m.editor = NewEditorModel()
	m.editor = m.editor.SetFocused(true)
	m.editor = m.editor.SetPrompt("\u276f ")
	m.editor = m.editor.SetPlaceholder("Try \"how does <filepath> work?\"")
	m.editor.width = m.width

	// Add user message to content
	um := NewUserMsgModel(text)
	m.content = append(m.content, um)

	// Add to conversation history
	m.messages = append(m.messages, ai.NewTextMessage(ai.RoleUser, text))

	// Check for slash commands
	if commands.IsCommand(text) {
		return m.handleSlashCommand(text)
	}

	// Start agent
	m.agentRunning = true
	return m, m.startAgentCmd()
}

func (m AppModel) handleSlashCommand(text string) (AppModel, tea.Cmd) {
	cmdCtx := &commands.CommandContext{
		Model:   m.modelName(),
		Mode:    m.mode.String(),
		Version: m.deps.Version,
		Messages: len(m.messages),
		ClearHistory: func() {
			// Will be handled via a message in a later phase
		},
		CompactFn: func() string {
			return "Compact not yet available."
		},
	}

	result, err := m.cmdRegistry.Dispatch(cmdCtx, text)
	if err != nil {
		result = fmt.Sprintf("Error: %v", err)
	}

	if result != "" {
		am := NewAssistantMsgModel()
		updated, _ := am.Update(AgentTextMsg{Text: result})
		m.content = append(m.content, updated)
	}
	return m, nil
}

func (m AppModel) startAgentCmd() tea.Cmd {
	program := m.sh.program
	deps := m.deps
	messages := make([]ai.Message, len(m.messages))
	copy(messages, m.messages)
	thinkingLevel := m.thinkingLevel

	return func() tea.Msg {
		if program == nil {
			return AgentErrorMsg{Err: fmt.Errorf("program reference not set")}
		}
		if deps.Provider == nil || deps.Model == nil {
			return AgentErrorMsg{Err: fmt.Errorf("no provider or model configured")}
		}

		// Build AI tools from agent tools
		aiTools := buildAITools(deps.Tools)
		llmCtx := &ai.Context{
			System:   deps.SystemPrompt,
			Messages: messages,
			Tools:    aiTools,
		}

		opts := &ai.StreamOptions{MaxTokens: 16384}
		if deps.Model.MaxOutputTokens > 0 {
			opts.MaxTokens = deps.Model.MaxOutputTokens
		}
		if thinkingLevel != config.ThinkingOff && deps.Model.SupportsThinking {
			opts.Thinking = true
		}

		// Create agent with permission checking
		permCheckFn := func(tool string, args map[string]any) error {
			if deps.Checker != nil {
				return deps.Checker.Check(tool, args)
			}
			return nil
		}

		ag := agent.NewWithPermissions(deps.Provider, deps.Model, deps.Tools, permCheckFn)
		events := ag.Prompt(context.Background(), llmCtx, opts)

		// The bridge sends all events via program.Send; blocks until done.
		RunAgentBridge(program, events)
		return nil // AgentDoneMsg already sent by RunAgentBridge
	}
}

// --- Internal helpers ---

func (m AppModel) propagateSize(msg tea.WindowSizeMsg) AppModel {
	for i := range m.content {
		updated, _ := m.content[i].Update(msg)
		m.content[i] = updated
	}
	updated, _ := m.editor.Update(msg)
	m.editor = updated.(EditorModel)
	fUpdated, _ := m.footer.Update(msg)
	m.footer = fUpdated.(FooterModel)
	return m
}

func (m AppModel) ensureAssistantMsg() AppModel {
	if len(m.content) == 0 {
		m.content = append(m.content, NewAssistantMsgModel())
		return m
	}
	if _, ok := m.content[len(m.content)-1].(AssistantMsgModel); !ok {
		m.content = append(m.content, NewAssistantMsgModel())
	}
	return m
}

func (m AppModel) updateLastAssistant(msg tea.Msg) AppModel {
	if len(m.content) == 0 {
		return m
	}
	idx := len(m.content) - 1
	updated, _ := m.content[idx].Update(msg)
	m.content[idx] = updated
	return m
}

func (m AppModel) updateOverlay(msg tea.Msg) (tea.Model, tea.Cmd) {
	updated, cmd := m.overlay.Update(msg)
	m.overlay = updated
	return m, cmd
}

func (m AppModel) toggleMode() AppModel {
	switch m.mode {
	case ModePlan:
		m.mode = ModeEdit
	case ModeEdit:
		m.mode = ModePlan
	}
	m.footer = m.footer.WithModeLabel(m.mode.String())
	return m
}

func (m AppModel) cycleThinking() AppModel {
	next := (m.thinkingLevel.Index() + 1) % 6
	m.thinkingLevel = config.ThinkingLevelFromIndex(next)
	m.footer = m.footer.WithThinking(m.thinkingLevel)
	return m
}

func (m AppModel) abortAgent() {
	if m.sh.activeAgent != nil {
		m.sh.activeAgent.Abort()
	}
}

func (m AppModel) modelName() string {
	if m.deps.Model != nil {
		return m.deps.Model.Name
	}
	return ""
}

func (m AppModel) buildCmdPalette() CmdPaletteModel {
	cmdList := m.cmdRegistry.List()
	entries := make([]CommandEntry, len(cmdList))
	for i, c := range cmdList {
		entries[i] = CommandEntry{Name: c.Name, Description: c.Description}
	}
	return NewCmdPaletteModel(entries)
}

// buildAITools converts AgentTool slice to ai.Tool slice for LLM context.
func buildAITools(tools []*agent.AgentTool) []ai.Tool {
	out := make([]ai.Tool, 0, len(tools))
	for _, t := range tools {
		schema := t.Parameters
		if schema == nil {
			schema = json.RawMessage(`{}`)
		}
		out = append(out, ai.Tool{
			Name:        t.Name,
			Description: t.Description,
			Parameters:  schema,
		})
	}
	return out
}

// detectGitBranch returns the current git branch name, or empty string.
func detectGitBranch() string {
	out, err := exec.Command("git", "rev-parse", "--abbrev-ref", "HEAD").Output()
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(out))
}

// detectGitCWD returns the current working directory for display.
func detectGitCWD() string {
	out, err := exec.Command("git", "rev-parse", "--show-toplevel").Output()
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(out))
}
