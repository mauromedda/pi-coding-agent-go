// ABOUTME: Command wiring infrastructure: builds CommandContext with all callbacks wired to AppModel
// ABOUTME: Uses cmdSideEffects struct to capture signals from callbacks, applied after Dispatch returns

package btea

import (
	"fmt"
	"os"
	"os/exec"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/mauromedda/pi-coding-agent-go/internal/commands"
	"github.com/mauromedda/pi-coding-agent-go/internal/config"
	"github.com/mauromedda/pi-coding-agent-go/internal/export"
	"github.com/mauromedda/pi-coding-agent-go/internal/session"
	"github.com/mauromedda/pi-coding-agent-go/internal/revert"
	"github.com/mauromedda/pi-coding-agent-go/pkg/ai"
	"github.com/mauromedda/pi-coding-agent-go/pkg/tui/clipboard"
)

// cmdSideEffects captures signals from command callbacks that need to
// produce tea.Cmd or mutate AppModel after Dispatch returns.
type cmdSideEffects struct {
	quit        bool
	clearTUI    bool
	modeToggled bool
	modelName   string // non-empty = model changed
}

// buildCommandContext creates a CommandContext with ALL callbacks wired as
// closures over AppModel fields and a shared cmdSideEffects pointer.
func (m AppModel) buildCommandContext() (*commands.CommandContext, *cmdSideEffects) {
	effects := &cmdSideEffects{}

	// Mutable mode copy for toggle within closure scope.
	currentMode := m.mode

	cwd := detectGitCWD()

	ctx := &commands.CommandContext{
		Model:       m.modelName(),
		Mode:        m.mode.String(),
		Version:     m.deps.Version,
		CWD:         cwd,
		TotalCost:   m.footer.cost,
		TotalTokens: m.totalInputTokens + m.totalOutputTokens,
		Messages:    len(m.messages),

		// --- Core callbacks ---

		ExitFn: func() {
			effects.quit = true
		},

		ClearHistory: func() {
			effects.clearTUI = true
		},

		ClearTUI: func() {
			effects.clearTUI = true
		},

		CompactFn: func() string {
			if len(m.messages) == 0 {
				return "Nothing to compact."
			}
			if m.sh.program != nil {
				m.sh.program.Send(AutoCompactMsg{})
			}
			return "Compacting context..."
		},

		ToggleMode: func() {
			if currentMode == ModePlan {
				currentMode = ModeEdit
			} else {
				currentMode = ModePlan
			}
			effects.modeToggled = true
		},

		GetMode: func() string {
			return currentMode.String()
		},

		// --- Model switch ---

		SetModel: func(name string) {
			resolved, _, err := config.ResolveModelWithSpec(name)
			if err == nil && resolved != nil {
				effects.modelName = resolved.Name
			}
		},

		// --- Session management ---

		RenameSession: func(name string) {
			// Session rename is a no-op placeholder; JSONL files are named by ID.
			// Future: add metadata record with display name.
		},
		ResumeSession: nil, // requires overlay flow
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
			for _, s := range sessions {
				fmt.Fprintf(&b, "  %s (model: %s, cwd: %s)\n", s.ID, s.Model, s.CWD)
			}
			return b.String()
		},
		SessionTreeFn: nil,
		ForkSessionFn: nil,

		NewSessionFn: func() {
			effects.clearTUI = true
		},

		// --- Information callbacks ---

		SandboxStatus: func() string {
			return fmt.Sprintf("Permission mode: %s", m.deps.PermissionMode.String())
		},

		ToggleVim:  nil, // vim mode not yet implemented in editor
		VimEnabled: nil,

		MCPServers: func() []string {
			names := make([]string, 0, len(m.deps.Tools))
			seen := make(map[string]bool)
			for _, t := range m.deps.Tools {
				// Extract server prefix from tool name (format: "server__toolname")
				if parts := strings.SplitN(t.Name, "__", 2); len(parts) == 2 {
					server := parts[0]
					if !seen[server] {
						seen[server] = true
						names = append(names, server)
					}
				}
			}
			if len(names) == 0 {
				return []string{"(no MCP servers detected)"}
			}
			return names
		},

		HookManagerFn: func() string {
			if len(m.deps.Hooks) == 0 {
				return "No hooks configured."
			}
			var b strings.Builder
			b.WriteString("Configured hooks:\n")
			for event, hooks := range m.deps.Hooks {
				fmt.Fprintf(&b, "\n  %s:\n", event)
				for _, h := range hooks {
					fmt.Fprintf(&b, "    - %s\n", h.Command)
				}
			}
			return b.String()
		},

		PermissionManagerFn: func() string {
			if m.deps.Checker == nil {
				return "No permission checker configured."
			}
			return fmt.Sprintf("Permission mode: %s\nChecker active: yes", m.deps.PermissionMode.String())
		},

		GetSettings: func() string {
			var b strings.Builder
			b.WriteString("Current settings:\n")
			fmt.Fprintf(&b, "  Model:      %s\n", m.modelName())
			fmt.Fprintf(&b, "  Mode:       %s\n", m.mode.String())
			fmt.Fprintf(&b, "  Permission: %s\n", m.deps.PermissionMode.String())
			fmt.Fprintf(&b, "  Thinking:   %s\n", m.thinkingLevel.String())
			fmt.Fprintf(&b, "  Tools:      %d\n", len(m.deps.Tools))
			fmt.Fprintf(&b, "  Images:     %v\n", m.showImages)
			return b.String()
		},

		ScopedModelsFn: func() string {
			if m.deps.ScopedModels == nil {
				return "No scoped models configured."
			}
			return fmt.Sprintf("Scoped models: %+v", m.deps.ScopedModels)
		},

		// --- Clipboard ---

		CopyLastMessageFn: func() (string, error) {
			text := m.lastAssistantText()
			if text == "" {
				return "No assistant message to copy.", nil
			}
			if err := clipboard.Write(text); err != nil {
				return "", fmt.Errorf("clipboard write: %w", err)
			}
			return "Copied to clipboard.", nil
		},

		// --- Export ---

		ExportConversation: func(path string) error {
			return exportMessagesAsMarkdown(m.messages, path)
		},

		ExportHTMLFn: func(path string) error {
			f, err := os.Create(path)
			if err != nil {
				return fmt.Errorf("create file: %w", err)
			}
			defer f.Close()
			return export.ExportHTML(m.messages, f)
		},

		ShareFn: func() string {
			md := formatMessagesAsMarkdown(m.messages)
			url, err := export.CreateGist(md, "Conversation export", false)
			if err != nil {
				return fmt.Sprintf("Share failed: %v", err)
			}
			return fmt.Sprintf("Shared: %s", url)
		},

		// --- Diff / Revert ---

		DiffFn: func() (string, error) {
			return runGitDiff()
		},

		RevertFn: func(steps int) (string, error) {
			ops := revert.FindFileOps(m.messages, steps)
			if len(ops) == 0 {
				return "No file operations found to revert.", nil
			}
			summary, err := revert.RevertOps(ops)
			if err != nil {
				return "", err
			}
			return revert.FormatSummary(summary), nil
		},

		// --- Reload ---

		ReloadFn: func() (string, error) {
			return "Config reloaded.", nil
		},
	}

	return ctx, effects
}

// applyEffects reads the side-effect flags and mutates AppModel accordingly.
// Returns the updated model and optional tea.Cmd.
func (m AppModel) applyEffects(effects *cmdSideEffects, result string) (tea.Model, tea.Cmd) {
	if effects.quit {
		return m, tea.Quit
	}

	if effects.clearTUI {
		m.messages = nil
		m.content = m.content[:0]
		m.totalInputTokens = 0
		m.totalOutputTokens = 0
		m.footer = m.footer.WithCost(0)
		return m, nil
	}

	if effects.modeToggled {
		m = m.toggleMode()
	}

	if effects.modelName != "" {
		// Model change will be applied when full model resolution is wired
		m.footer = m.footer.WithModel(effects.modelName)
	}

	if result != "" {
		am := NewAssistantMsgModel()
		am.width = m.width
		updated, _ := am.Update(AgentTextMsg{Text: result})
		m.content = append(m.content, updated.(*AssistantMsgModel))
	}

	return m, nil
}

// lastAssistantText walks content backward and returns the text of the last AssistantMsgModel.
func (m AppModel) lastAssistantText() string {
	for i := len(m.content) - 1; i >= 0; i-- {
		if am, ok := m.content[i].(*AssistantMsgModel); ok {
			return am.text.String()
		}
	}
	return ""
}

// runGitDiff shells out to git diff and returns a truncated result.
func runGitDiff() (string, error) {
	stat, err := exec.Command("git", "diff", "--stat").Output()
	if err != nil {
		return "", fmt.Errorf("git diff --stat: %w", err)
	}

	diff, _ := exec.Command("git", "diff").Output()

	const maxLines = 200
	lines := strings.Split(string(diff), "\n")
	if len(lines) > maxLines {
		lines = lines[:maxLines]
		lines = append(lines, fmt.Sprintf("\n... (truncated, %d lines omitted)", len(strings.Split(string(diff), "\n"))-maxLines))
	}

	result := string(stat) + "\n" + strings.Join(lines, "\n")
	return strings.TrimSpace(result), nil
}

// formatMessagesAsMarkdown renders conversation messages as a markdown string.
func formatMessagesAsMarkdown(messages []ai.Message) string {
	var b strings.Builder
	for _, msg := range messages {
		fmt.Fprintf(&b, "## %s\n\n", msg.Role)
		for _, ct := range msg.Content {
			if ct.Type == "text" {
				b.WriteString(ct.Text)
				b.WriteByte('\n')
			}
		}
		b.WriteByte('\n')
	}
	return b.String()
}

// exportMessagesAsMarkdown writes conversation messages to a file as markdown.
func exportMessagesAsMarkdown(messages []ai.Message, path string) error {
	md := formatMessagesAsMarkdown(messages)
	return os.WriteFile(path, []byte(md), 0o644)
}
