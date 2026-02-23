// ABOUTME: Root AppModel wiring all sub-models for the Bubble Tea interactive TUI
// ABOUTME: Handles message routing, overlay management, agent lifecycle, and key dispatch

package btea

import (
	"context"
	"encoding/json"
	"fmt"
	"os/exec"
	"strings"
	"sync/atomic"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/mauromedda/pi-coding-agent-go/internal/agent"
	"github.com/mauromedda/pi-coding-agent-go/internal/commands"
	"github.com/mauromedda/pi-coding-agent-go/internal/config"
	"github.com/mauromedda/pi-coding-agent-go/internal/perf"
	"github.com/mauromedda/pi-coding-agent-go/internal/permission"
	"github.com/mauromedda/pi-coding-agent-go/internal/session"
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
	activeAgent atomic.Pointer[agent.Agent]
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
	modelProfile  *perf.ModelProfile

	// Image display
	showImages bool

	// Command handling
	cmdRegistry *commands.Registry

	// Prompt queue and history
	promptQueue   []string // prompts waiting to run after current agent finishes
	promptHistory []string // all submitted prompts (most recent last)
	historyIndex  int      // -1 = composing new; 0+ = browsing history (0 = most recent)
	savedDraft    string   // editor text saved before entering history mode

	// Compaction state
	compacting bool

	// Retry state
	retryCount int       // number of retries attempted for current error
	retryAt    time.Time // when to retry next

	// Async bash state
	bashRunning bool

	// Git working directory (populated async in Init)
	gitCWD string

	// Cached separator string (recomputed only on WindowSizeMsg)
	cachedSep string
}

// Compile-time interface assertion.
var _ tea.Model = AppModel{}

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
		WithPath("").
		WithModel(modelName).
		WithModeLabel(initialMode.String()).
		WithPermissionMode(permLabel).
		WithShowImages(true)

	welcome := NewWelcomeModel(deps.Version, modelName, "", toolCount)

	return AppModel{
		sh:           &shared{ctx: ctx, cancel: cancel},
		mode:         initialMode,
		editor:       editor,
		footer:       footer,
		content:      []tea.Model{welcome},
		deps:         deps,
		cmdRegistry:  commands.NewRegistry(),
		showImages:   true,
		historyIndex: -1,
	}
}

// Init returns startup commands: detect git branch, git CWD, and probe model latency.
func (m AppModel) Init() tea.Cmd {
	gitBranchCmd := func() tea.Msg {
		return gitBranchMsg{branch: detectGitBranch()}
	}

	gitCWDCmd := func() tea.Msg {
		return gitCWDMsg{cwd: detectGitCWD()}
	}

	probeCmd := func() tea.Msg {
		if m.deps.Model == nil {
			return nil
		}

		// Skip probe for known-remote APIs (always reachable, no localhost dial).
		switch m.deps.Model.Api {
		case ai.ApiAnthropic, ai.ApiGoogle:
			profile := perf.BuildProfile(m.deps.Model, perf.ProbeResult{
				TTFB:    300 * time.Millisecond,
				Latency: perf.LatencyFast,
			})
			return ProbeResultMsg{Profile: profile}
		}

		baseURL := m.deps.Model.BaseURL
		if baseURL == "" {
			// Use a reasonable default; probe will classify as Slow on error.
			baseURL = "http://localhost:11434" // ollama default
		}
		probe := perf.ProbeTTFB(context.Background(), baseURL, "")
		profile := perf.BuildProfile(m.deps.Model, probe)
		return ProbeResultMsg{Profile: profile}
	}

	return tea.Batch(gitBranchCmd, gitCWDCmd, probeCmd)
}

// Update routes messages to the appropriate handler.
func (m AppModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	// --- Layout ---
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.cachedSep = strings.Repeat("─", msg.Width)
		m = m.propagateSize(msg)
		// Propagate to overlay so it can track width/height
		if m.overlay != nil {
			updated, _ := m.overlay.Update(msg)
			m.overlay = updated
		}
		return m, nil

	// --- Overlay lifecycle ---
	case DismissOverlayMsg:
		m.overlay = nil
		return m, nil

	// --- Overlay result messages (always handled by root, even when overlay is active) ---
	case CmdPaletteSelectMsg:
		m.overlay = nil
		// Place command text in editor for user to review/submit (not auto-submit)
		m.editor = m.editor.SetFocused(true).SetText("/" + msg.Name)
		return m, nil

	case CmdPaletteDismissMsg:
		m.overlay = nil
		m.editor = m.editor.SetFocused(true)
		return m, nil

	case FileMentionSelectMsg:
		m.overlay = nil
		text := m.editor.Text()
		if text != "" && !strings.HasSuffix(text, " ") {
			text += " "
		}
		text += "@" + msg.RelPath
		m.editor = m.editor.SetFocused(true).SetText(text)
		return m, nil

	case FileMentionDismissMsg:
		m.overlay = nil
		m.editor = m.editor.SetFocused(true)
		return m, nil

	case ModelSelectedMsg:
		m.overlay = nil
		m.editor = m.editor.SetFocused(true)
		// Apply model switch
		if m.deps.Model == nil {
			m.deps.Model = &ai.Model{}
		}
		m.deps.Model.Name = msg.Model.Name
		m.deps.Model.ID = msg.Model.ID
		m.footer = m.footer.WithModel(msg.Model.Name)
		return m, nil

	case ModelSelectorDismissMsg:
		m.overlay = nil
		m.editor = m.editor.SetFocused(true)
		return m, nil

	case SessionSelectedMsg:
		m.overlay = nil
		m.editor = m.editor.SetFocused(true)
		// Placeholder: session resume will be wired in a later phase
		return m, nil

	case SessionSelectorDismissMsg:
		m.overlay = nil
		m.editor = m.editor.SetFocused(true)
		return m, nil

	// --- Queue overlay results ---
	case QueueUpdatedMsg:
		m.overlay = nil
		m.promptQueue = msg.Items
		m.footer = m.footer.WithQueuedCount(len(m.promptQueue))
		m.editor = m.editor.SetFocused(true)
		// Resume drain if agent finished while overlay was open
		if !m.agentRunning && len(m.promptQueue) > 0 {
			next := m.promptQueue[0]
			m.promptQueue = m.promptQueue[1:]
			m.footer = m.footer.WithQueuedCount(len(m.promptQueue))
			return m.submitPrompt(next)
		}
		return m, nil

	case QueueEditMsg:
		m.overlay = nil
		// Remove the item from queue
		if msg.Index >= 0 && msg.Index < len(m.promptQueue) {
			m.promptQueue = append(m.promptQueue[:msg.Index], m.promptQueue[msg.Index+1:]...)
		}
		m.footer = m.footer.WithQueuedCount(len(m.promptQueue))
		m.editor = m.editor.SetFocused(true).SetText(msg.Text)
		return m, nil

	// --- Agent done (must be handled regardless of overlay) ---
	case AgentDoneMsg:
		m.agentRunning = false
		if len(msg.Messages) > 0 {
			// Persist new assistant messages to session
			if m.deps.Session != nil {
				for _, am := range msg.Messages {
					if am.Role == ai.RoleAssistant {
						m.deps.Session.AddAssistantMessage(&ai.AssistantMessage{
							Content: am.Content,
						})
					}
				}
			}
			m.messages = msg.Messages
		}
		// Drain next queued prompt; skip if queue overlay is open (user is editing)
		if _, editing := m.overlay.(QueueViewModel); !editing && len(m.promptQueue) > 0 {
			next := m.promptQueue[0]
			m.promptQueue = m.promptQueue[1:]
			m.footer = m.footer.WithQueuedCount(len(m.promptQueue))
			return m.submitPrompt(next)
		}
		return m, nil

	// --- Plan overlay results ---
	case PlanApprovedMsg:
		m.overlay = nil
		// Plan approved; could trigger execute mode in future phases
		return m, nil

	case PlanRejectedMsg:
		m.overlay = nil
		return m, nil

	// --- Permission flow ---
	case PermissionRequestMsg:
		m.overlay = NewPermDialogModel(msg.Tool, msg.Args, msg.ReplyCh)
		return m, nil

	default:
		// Route to overlay if active (key presses, etc.)
		if m.overlay != nil {
			// When command palette is active, mirror typed/deleted chars to editor
			if _, isPalette := m.overlay.(CmdPaletteModel); isPalette {
				if keyMsg, isKey := msg.(tea.KeyMsg); isKey {
					if keyMsg.Type == tea.KeyRunes || keyMsg.Type == tea.KeyBackspace {
						editorUpdated, _ := m.editor.Update(keyMsg)
						m.editor = editorUpdated.(EditorModel)
					}
				}
			}
			return m.updateOverlay(msg)
		}
	}

	// Non-overlay messages (only reached when no overlay is active)
	switch msg := msg.(type) {
	case gitBranchMsg:
		m.gitBranch = msg.branch
		m.footer = m.footer.WithGitBranch(msg.branch)
		return m, nil

	case gitCWDMsg:
		m.gitCWD = msg.cwd
		m.footer = m.footer.WithPath(msg.cwd)
		// Update welcome model if it's the first content item
		if len(m.content) > 0 {
			if _, ok := m.content[0].(WelcomeModel); ok {
				m.content[0] = NewWelcomeModel(m.deps.Version, m.modelName(), msg.cwd, len(m.deps.Tools))
			}
		}
		return m, nil

	case ProbeResultMsg:
		m.modelProfile = &msg.Profile
		m.footer = m.footer.WithLatencyClass(msg.Profile.Latency.String())
		return m, nil

	case SessionLoadedMsg:
		m.messages = msg.Messages
		// Rebuild content from loaded messages
		for _, am := range msg.Messages {
			switch am.Role {
			case ai.RoleUser:
				text := ""
				for _, c := range am.Content {
					if c.Type == "text" {
						text += c.Text
					}
				}
				m.content = append(m.content, NewUserMsgModel(text))
			case ai.RoleAssistant:
				assistantModel := NewAssistantMsgModel()
				assistantModel.width = m.width
				text := ""
				for _, c := range am.Content {
					if c.Type == "text" {
						text += c.Text
					}
				}
				updated, _ := assistantModel.Update(AgentTextMsg{Text: text})
				m.content = append(m.content, updated.(*AssistantMsgModel))
			}
		}
		return m, nil

	case SessionSavedMsg:
		return m, nil

	case AutoCompactMsg:
		return m.autoCompact()

	case CompactDoneMsg:
		m.compacting = false
		if len(msg.Messages) > 0 {
			m.messages = msg.Messages
		}
		// Persist compaction to session if wired
		if m.deps.Session != nil && m.deps.Session.Writer != nil {
			_ = m.deps.Session.Writer.WriteCompaction(session.CompactionData{
				Summary:      msg.Summary,
				TokensBefore: msg.TokensSaved,
			})
		}
		return m, nil

	// --- Phase 8: TUI enhancement messages ---
	case ModeTransitionMsg:
		m.footer = m.footer.WithIntentLabel(msg.To)
		return m, nil

	case SettingsChangedMsg:
		// Re-render footer; more wiring in Phase 9
		return m, nil

	case PlanGeneratedMsg:
		m.overlay = NewPlanViewModel(msg.Plan)
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

		// Update context window usage percentage
		if m.deps.Model != nil {
			ctxWindow := m.deps.Model.EffectiveContextWindow()
			if ctxWindow > 0 {
				pct := m.totalInputTokens * 100 / ctxWindow
				if pct > 100 {
					pct = 100
				}
				m.footer = m.footer.WithContextPct(pct)
			}
		}

		// Check if auto-compaction should trigger
		threshold := m.deps.AutoCompactThreshold
		if threshold > 0 && !m.compacting {
			total := m.totalInputTokens + m.totalOutputTokens
			if total > threshold {
				return m, func() tea.Msg { return AutoCompactMsg{} }
			}
		}
		return m, nil

	case AgentErrorMsg:
		// Check for rate-limit errors and auto-retry
		if isRateLimited(msg.Err) && m.retryCount < maxRetries {
			m.retryCount++
			backoff := retryBackoff(m.retryCount)
			m.retryAt = time.Now().Add(backoff)
			return m, tea.Tick(time.Second, func(t time.Time) tea.Msg {
				remaining := time.Until(m.retryAt)
				if remaining < 0 {
					remaining = 0
				}
				return RetryTickMsg{Remaining: remaining}
			})
		}
		m = m.ensureAssistantMsg()
		m = m.updateLastAssistant(msg)
		return m, nil

	case RetryTickMsg:
		if msg.Remaining > 0 {
			// Keep ticking
			return m, tea.Tick(time.Second, func(t time.Time) tea.Msg {
				remaining := time.Until(m.retryAt)
				if remaining < 0 {
					remaining = 0
				}
				return RetryTickMsg{Remaining: remaining}
			})
		}
		// Timer expired; restart the agent
		m.agentRunning = true
		return m, m.startAgentCmd()

	case BashDoneMsg:
		m.bashRunning = false
		bom := NewBashOutputModel(msg.Command)
		bom.AddOutput(msg.Output)
		bom.SetExitCode(msg.ExitCode)
		bom.width = m.width
		m.content = append(m.content, bom)
		return m, nil

	case AgentCancelMsg:
		m = m.ensureAssistantMsg()
		m = m.updateLastAssistant(AgentTextMsg{Text: "\n⏹ Agent cancelled."})
		return m, nil

	// --- Key routing ---
	case tea.KeyMsg:
		return m.handleKey(msg)
	}

	return m, nil
}

// maxVisibleContent is the maximum number of content models rendered in View().
// Older models are skipped to bound string allocations in long sessions.
const maxVisibleContent = 50

// View renders the full TUI layout.
func (m AppModel) View() string {
	var sections []string

	// Only render the last N content models to avoid unbounded allocations.
	start := max(len(m.content)-maxVisibleContent, 0)
	for _, c := range m.content[start:] {
		sections = append(sections, c.View())
	}

	s := Styles()

	// Determine separator color based on editor text and last content
	sepColor := s.Border
	if !m.editor.IsEmpty() && strings.HasPrefix(m.editor.Text(), "!") {
		sepColor = s.BashSeparator
	} else if len(m.content) > 0 {
		if _, isAssistant := m.content[len(m.content)-1].(*AssistantMsgModel); isAssistant {
			sepColor = s.BashSeparator
		} else if _, isBashOutput := m.content[len(m.content)-1].(*BashOutputModel); isBashOutput {
			sepColor = s.BashSeparator
		}
	}

	// Dropdown overlays (command palette, file mention) render inline
	// above the editor rather than centered on screen.
	if m.overlay != nil && isDropdownOverlay(m.overlay) {
		dropdownView := m.overlay.View()
		if dropdownView != "" {
			sections = append(sections, dropdownView)
		}
	}

	// Use cached separator string (recomputed only on WindowSizeMsg)
	sep := m.cachedSep
	sections = append(sections,
		sepColor.Render(sep),
		m.editor.View(),
	)

	sections = append(sections,
		s.Border.Render(sep),
		m.footer.View(),
	)

	main := lipgloss.JoinVertical(lipgloss.Left, sections...)

	// Centered overlays (permission dialog, cost view, plan view, etc.)
	if m.overlay != nil && !isDropdownOverlay(m.overlay) {
		return overlayRender(main, m.overlay.View(), m.width, m.height)
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

	case "esc":
		if m.agentRunning {
			m.abortAgent()
			return m, func() tea.Msg { return AgentCancelMsg{} }
		}
		return m, nil

	case "ctrl+l":
		// Clear viewport; keep only a fresh welcome
		m.content = m.content[:0]
		welcome := NewWelcomeModel(m.deps.Version, m.modelName(), m.gitCWD, len(m.deps.Tools))
		m.content = append(m.content, welcome)
		return m, nil

	case "shift+tab":
		m = m.toggleMode()
		return m, nil

	case "alt+t":
		m = m.cycleThinking()
		return m, nil

	case "ctrl+t":
		// Toggle cost dashboard
		if m.overlay != nil {
			m.overlay = nil
		} else {
			m.overlay = NewCostViewModel(
				m.totalInputTokens, m.totalOutputTokens, 0,
				m.footer.cost, 0, 0,
			)
		}
		return m, nil

	case "alt+m":
		// Open model selector overlay
		m.overlay = NewModelSelectorModel(m.deps.AvailableModels)
		return m, nil

	case "alt+i":
		m.showImages = !m.showImages
		m.footer = m.footer.WithShowImages(m.showImages)
		// Propagate toggle to all content models
		toggleMsg := ToggleImagesMsg{Show: m.showImages}
		for i := range m.content {
			updated, _ := m.content[i].Update(toggleMsg)
			m.content[i] = updated
		}
		return m, nil

	case "ctrl+e":
		if len(m.promptQueue) > 0 {
			m.overlay = NewQueueViewModel(m.promptQueue, m.width)
			return m, nil
		}
		// No queue: fall through to editor (end-of-line)
		updated, cmd := m.editor.Update(msg)
		m.editor = updated.(EditorModel)
		return m, cmd

	case "ctrl+o":
		// Propagate to content models so ToolCallModel can toggle expand/collapse
		for i := range m.content {
			updated, _ := m.content[i].Update(msg)
			m.content[i] = updated
		}
		return m, nil

	case "enter":
		if !m.editor.IsEmpty() {
			text := m.editor.Text()
			if m.agentRunning {
				// Enqueue for later; history is populated when drain calls submitPrompt
				m.promptQueue = append(m.promptQueue, text)
				m.historyIndex = -1
				m.savedDraft = ""
				m.editor = m.resetEditor()
				m.footer = m.footer.WithQueuedCount(len(m.promptQueue))
				return m, nil
			}
			return m.submitPrompt(text)
		}
		// Let editor handle enter for multi-line
		updated, cmd := m.editor.Update(msg)
		m.editor = updated.(EditorModel)
		return m, cmd

	case "tab":
		// Tab accepts ghost text when no overlay is open
		if m.overlay == nil && m.editor.GhostText() != "" {
			updated, cmd := m.editor.Update(msg)
			m.editor = updated.(EditorModel)
			m.editor = m.editor.SetGhostText("")
			return m, cmd
		}
		// Otherwise, pass tab to overlay or editor
		if m.overlay != nil {
			return m.updateOverlay(msg)
		}
		updated, cmd := m.editor.Update(msg)
		m.editor = updated.(EditorModel)
		return m, cmd

	default:
		// Up arrow: history navigation when cursor is on first line
		if msg.Type == tea.KeyUp && !m.agentRunning && m.editor.CursorRow() == 0 {
			if len(m.promptHistory) > 0 {
				if m.historyIndex == -1 {
					m.savedDraft = m.editor.Text()
				}
				newIdx := m.historyIndex + 1
				if newIdx < len(m.promptHistory) {
					m.historyIndex = newIdx
					prompt := m.promptHistory[len(m.promptHistory)-1-newIdx]
					m.editor = m.editor.SetText(prompt)
					return m, nil
				}
			}
			return m, nil
		}

		// Down arrow: exit history when browsing
		if msg.Type == tea.KeyDown && m.historyIndex >= 0 {
			m.historyIndex--
			if m.historyIndex == -1 {
				m.editor = m.editor.SetText(m.savedDraft)
			} else {
				prompt := m.promptHistory[len(m.promptHistory)-1-m.historyIndex]
				m.editor = m.editor.SetText(prompt)
			}
			return m, nil
		}

		// Check for "/" to open command palette
		if msg.Type == tea.KeyRunes && len(msg.Runes) == 1 {
			switch msg.Runes[0] {
			case '/':
				if !m.agentRunning {
					// Keep "/" in editor and open command palette
					m.editor = m.editor.SetText("/")
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
		// Compute ghost text after each editor update
		m.editor = m.editor.SetGhostText(m.computeGhostText())
		return m, cmd
	}
}

// --- Prompt submission ---

func (m AppModel) submitPrompt(text string) (AppModel, tea.Cmd) {
	m.editor = m.resetEditor()

	// Track history
	m.promptHistory = append(m.promptHistory, text)
	m.historyIndex = -1
	m.savedDraft = ""

	// Add user message to content
	um := NewUserMsgModel(text)
	m.content = append(m.content, um)

	// Add to conversation history
	m.messages = append(m.messages, ai.NewTextMessage(ai.RoleUser, text))

	// Persist user message to session (if wired)
	if m.deps.Session != nil {
		_ = m.deps.Session.AddUserMessage(text)
	}

	// Check for commands
	if commands.IsCommand(text) {
		// Handle bash commands (starting with !)
		if text[0] == '!' {
			return m.handleBashCommand(text[1:])
		}
		// Handle slash commands
		return m.handleSlashCommand(text)
	}

	// Start agent
	m.agentRunning = true
	return m, m.startAgentCmd()
}

func (m AppModel) handleBashCommand(command string) (AppModel, tea.Cmd) {
	m.bashRunning = true
	cmd := command
	return m, func() tea.Msg {
		output, exitCode := runBashCommand(cmd)
		return BashDoneMsg{
			Command:  cmd,
			Output:   output,
			ExitCode: exitCode,
		}
	}
}

func runBashCommand(command string) (string, int) {
	// Use /bin/bash with full path to avoid PATH issues
	cmd := exec.Command("/bin/bash", "-c", command)
	output, err := cmd.CombinedOutput()
	if err != nil {
		exitCode := 1
		if exitErr, ok := err.(*exec.ExitError); ok {
			exitCode = exitErr.ExitCode()
		}
		return string(output), exitCode
	}
	return string(output), 0
}

func (m AppModel) handleSlashCommand(text string) (AppModel, tea.Cmd) {
	ctx, effects := m.buildCommandContext()

	result, err := m.cmdRegistry.Dispatch(ctx, text)
	if err != nil {
		result = fmt.Sprintf("Error: %v", err)
	}

	model, cmd := m.applyEffects(effects, result)
	return model.(AppModel), cmd
}

func (m AppModel) startAgentCmd() tea.Cmd {
	program := m.sh.program
	sh := m.sh // shared pointer for agent assignment
	deps := m.deps
	messages := make([]ai.Message, len(m.messages))
	copy(messages, m.messages)
	thinkingLevel := m.thinkingLevel
	profile := m.modelProfile

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
		sh.activeAgent.Store(ag) // enable cancellation via abortAgent()

		// Wire adaptive performance if probe has completed
		if profile != nil {
			ag.SetAdaptive(&agent.AdaptiveConfig{
				Profile: *profile,
			})
		}

		// Per-agent child context: cancelled when agent completes to prevent goroutine leaks.
		agCtx, agCancel := context.WithCancel(sh.ctx)
		defer agCancel()

		events := ag.Prompt(agCtx, llmCtx, opts)

		// The bridge sends streaming events via program.Send; blocks until done.
		RunAgentBridge(program, events)

		sh.activeAgent.Store(nil) // clear agent reference after completion
		// Return AgentDoneMsg with the updated conversation messages.
		return AgentDoneMsg{Messages: llmCtx.Messages}
	}
}

// --- Retry ---

const maxRetries = 3

// retryBackoff returns the backoff duration for the given retry attempt (1-based).
func retryBackoff(attempt int) time.Duration {
	switch attempt {
	case 1:
		return 2 * time.Second
	case 2:
		return 4 * time.Second
	default:
		return 8 * time.Second
	}
}

// isRateLimited returns true if the error looks like a rate-limit or overload.
func isRateLimited(err error) bool {
	if err == nil {
		return false
	}
	msg := strings.ToLower(err.Error())
	return strings.Contains(msg, "rate limit") ||
		strings.Contains(msg, "429") ||
		strings.Contains(msg, "overloaded")
}

// --- Compaction ---

// autoCompact starts an asynchronous compaction of the conversation context.
// Returns a no-op if already compacting or there are no messages.
func (m AppModel) autoCompact() (tea.Model, tea.Cmd) {
	if m.compacting || len(m.messages) == 0 {
		return m, nil
	}
	m.compacting = true

	messages := make([]ai.Message, len(m.messages))
	copy(messages, m.messages)

	cfg := session.CompactionConfig{
		ReserveTokens:    4096,
		KeepRecentTokens: 2048,
	}

	return m, func() tea.Msg {
		tokensBefore := session.EstimateMessagesTokens(messages)

		// Use a simple extractive summarizer (no LLM call) for now.
		// Future: inject LLM-based summarizer via deps.
		result, err := session.CompactWithLLM(
			context.Background(),
			messages,
			cfg,
			func(_ context.Context, msgs []ai.Message, _ string) (string, error) {
				// Simple extractive summary: concatenate first 500 chars of each message
				var b strings.Builder
				for _, msg := range msgs {
					for _, c := range msg.Content {
						if c.Type == "text" && c.Text != "" {
							text := c.Text
							if len(text) > 200 {
								text = text[:200] + "..."
							}
							fmt.Fprintf(&b, "[%s] %s\n", msg.Role, text)
						}
					}
				}
				return b.String(), nil
			},
		)
		if err != nil {
			return AgentErrorMsg{Err: fmt.Errorf("compaction: %w", err)}
		}

		tokensAfter := session.EstimateMessagesTokens(result.Messages)
		return CompactDoneMsg{
			Messages:    result.Messages,
			Summary:     result.Summary,
			TokensSaved: tokensBefore - tokensAfter,
		}
	}
}

// --- Internal helpers ---

// isDropdownOverlay returns true for overlays that should render inline
// above the editor (autocomplete dropdowns), false for overlays that
// should render centered on screen (dialogs, dashboards).
func isDropdownOverlay(overlay tea.Model) bool {
	switch overlay.(type) {
	case CmdPaletteModel, FileMentionModel:
		return true
	default:
		return false
	}
}

func (m AppModel) propagateSize(msg tea.WindowSizeMsg) AppModel {
	for i := range m.content {
		updated, _ := m.content[i].Update(msg)
		// Handle the case where Update returns a pointer (AssistantMsgModel)
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
		am := NewAssistantMsgModel()
		am.width = m.width
		m.content = append(m.content, am)
		return m
	}
	if _, ok := m.content[len(m.content)-1].(*AssistantMsgModel); !ok {
		am := NewAssistantMsgModel()
		am.width = m.width
		m.content = append(m.content, am)
	}
	return m
}

func (m AppModel) updateLastAssistant(msg tea.Msg) AppModel {
	if len(m.content) == 0 {
		return m
	}
	idx := len(m.content) - 1
	if _, ok := m.content[idx].(*AssistantMsgModel); !ok {
		return m
	}
	updated, _ := m.content[idx].Update(msg)
	m.content[idx] = updated.(*AssistantMsgModel)
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
	if ag := m.sh.activeAgent.Load(); ag != nil {
		ag.Abort()
	}
}

// resetEditor creates a fresh editor with standard configuration (focused, prompt, placeholder, width).
func (m AppModel) resetEditor() EditorModel {
	e := NewEditorModel()
	e = e.SetFocused(true)
	e = e.SetPrompt("\u276f ")
	e = e.SetPlaceholder("Try \"how does <filepath> work?\"")
	e.width = m.width
	return e
}

func (m AppModel) modelName() string {
	if m.deps.Model != nil {
		return m.deps.Model.Name
	}
	return ""
}

// computeGhostText returns the completion suffix for the current editor text.
// Only active when text starts with "/" and has no spaces.
func (m AppModel) computeGhostText() string {
	text := m.editor.Text()
	if !strings.HasPrefix(text, "/") || strings.Contains(text, " ") {
		return ""
	}
	prefix := text[1:] // strip "/"
	if prefix == "" {
		return ""
	}
	match := m.cmdRegistry.BestMatch(prefix)
	if match == "" {
		return ""
	}
	// Return the suffix that would complete the command
	return match[len(prefix):]
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
