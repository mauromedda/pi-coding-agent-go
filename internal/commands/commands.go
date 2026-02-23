// ABOUTME: Slash command registry and dispatch for interactive mode
// ABOUTME: Provides categorized slash commands with nilable callback pattern for extensibility

package commands

import (
	"fmt"
	"path/filepath"
	"sort"
	"strings"

	"github.com/mauromedda/pi-coding-agent-go/internal/changelog"
)

// Command represents a slash command.
type Command struct {
	Name        string
	Aliases     []string // shorthand aliases (e.g., "q" for "quit")
	Category    string
	Description string
	Execute     func(ctx *CommandContext, args string) (string, error)
}

// CommandContext provides access to app state for commands.
type CommandContext struct {
	Model        string
	Mode         string
	Version      string
	CWD          string
	TotalCost    float64
	TotalTokens  int
	Messages     int
	SetModel     func(string)
	ClearHistory func()
	CompactFn    func() string

	// Exit callback. Nilable; /exit returns "not available" when nil.
	ExitFn func()

	// ClearTUI clears the visual TUI (screen + components). Nilable.
	ClearTUI func()

	// Extended command callbacks. All nilable; commands return "not available" when nil.
	MemoryEntries      []string
	ToggleMode         func()
	GetMode            func() string
	RenameSession      func(string)
	ResumeSession      func(string) error
	SandboxStatus      func() string
	ToggleVim          func()
	VimEnabled         func() bool
	MCPServers         func() []string
	ExportConversation func(string) error
	ReloadFn           func() (string, error)

	// Phase 1 integration callbacks
	SessionTreeFn       func() string // /tree: show interactive session tree
	HookManagerFn       func() string // /hooks: show hook manager
	PermissionManagerFn func() string // /permissions: show permission manager
	ScopedModelsFn      func() string // /scoped-models: show model config
	KeybindingsFn       func() string // /hotkeys: show keybindings
	ListSessionsFn      func() string // /resume with no args: list sessions

	// Session management callbacks
	CopyLastMessageFn func() (string, error) // /copy: copy last assistant message to clipboard
	NewSessionFn      func()                 // /new: start new session
	ForkSessionFn     func() (string, error) // /fork: fork current session

	// Phase 4 integration callbacks
	GetSettings  func() string       // /settings: show current settings
	ShareFn      func() string       // /share: share current session
	ExportHTMLFn func(string) error  // /export <path>.html: HTML export handler
}

// Registry holds all registered slash commands.
type Registry struct {
	commands map[string]*Command
}

// NewRegistry creates a registry with all core commands registered.
func NewRegistry() *Registry {
	r := &Registry{commands: make(map[string]*Command)}
	r.registerCoreCommands()
	r.registerAliases()
	return r
}

// registerAliases adds alias entries pointing to the same Command.
// Must be called after registerCoreCommands.
func (r *Registry) registerAliases() {
	for _, cmd := range r.List() {
		for _, alias := range cmd.Aliases {
			r.commands[alias] = cmd
		}
	}
}

// Get returns a command by name.
// The second return value indicates whether the name was found.
func (r *Registry) Get(name string) (*Command, bool) {
	cmd, ok := r.commands[name]
	return cmd, ok
}

// List returns all commands sorted by name for deterministic output.
// Alias entries (where the map key differs from cmd.Name) are excluded.
func (r *Registry) List() []*Command {
	result := make([]*Command, 0, len(r.commands))
	for key, cmd := range r.commands {
		if key == cmd.Name {
			result = append(result, cmd)
		}
	}
	sort.Slice(result, func(i, j int) bool {
		return result[i].Name < result[j].Name
	})
	return result
}

// Dispatch parses a "/command args" input, looks up the command, and executes it.
// Returns the command output or an error if the command is not found.
func (r *Registry) Dispatch(ctx *CommandContext, input string) (string, error) {
	input = strings.TrimSpace(input)
	if !IsCommand(input) {
		return "", fmt.Errorf("not a command: %q", input)
	}

	// Strip leading '/' and split into command name + args.
	raw := input[1:]
	parts := strings.SplitN(raw, " ", 2)
	name := parts[0]
	args := ""
	if len(parts) > 1 {
		args = strings.TrimSpace(parts[1])
	}

	cmd, ok := r.commands[name]
	if !ok {
		return "", fmt.Errorf("unknown command: /%s", name)
	}
	return cmd.Execute(ctx, args)
}

// IsCommand returns true if input starts with '/' or '!'.
func IsCommand(input string) bool {
	return len(input) > 0 && (input[0] == '/' || input[0] == '!')
}

// defaultHotkeysTable returns a formatted table of default keybindings.
func defaultHotkeysTable() string {
	return `Keyboard shortcuts:

  Ctrl+C        Abort / Exit
  Ctrl+D        Exit
  Ctrl+G        Open external editor
  Shift+Tab     Toggle Plan/Edit mode
  Enter         Send message
  @             File mention autocomplete
  Alt+Enter     Queue follow-up message
  Alt+Up/Down   Cycle message history`
}

// registerCoreCommands adds all built-in slash commands to the registry.
func (r *Registry) registerCoreCommands() {
	core := []*Command{
		{
			Name:        "clear",
			Category:    "Session",
			Description: "Clear conversation history",
			Execute: func(ctx *CommandContext, _ string) (string, error) {
				ctx.ClearHistory()
				if ctx.ClearTUI != nil {
					ctx.ClearTUI()
				}
				return "Conversation cleared.", nil
			},
		},
		{
			Name:        "compact",
			Category:    "Session",
			Description: "Compact conversation into a summary",
			Execute: func(ctx *CommandContext, _ string) (string, error) {
				return ctx.CompactFn(), nil
			},
		},
		{
			Name:        "config",
			Category:    "Config",
			Description: "Show current configuration",
			Execute: func(ctx *CommandContext, _ string) (string, error) {
				return fmt.Sprintf(
					"Model:   %s\nMode:    %s\nCWD:     %s\nVersion: %s",
					ctx.Model, ctx.Mode, ctx.CWD, ctx.Version,
				), nil
			},
		},
		{
			Name:        "help",
			Aliases:     []string{"h", "?"},
			Category:    "Info",
			Description: "Show available commands",
			Execute: func(_ *CommandContext, _ string) (string, error) {
				// Group commands by category
				categories := map[string][]*Command{}
				categoryOrder := []string{"Session", "Mode", "Config", "Info"}
				for _, cmd := range r.List() {
					cat := cmd.Category
					if cat == "" {
						cat = "Info"
					}
					categories[cat] = append(categories[cat], cmd)
				}
				var b strings.Builder
				b.WriteString("Available commands:\n")
				for _, cat := range categoryOrder {
					cmds := categories[cat]
					if len(cmds) == 0 {
						continue
					}
					fmt.Fprintf(&b, "\n## %s\n", cat)
					for _, cmd := range cmds {
						fmt.Fprintf(&b, "  /%s — %s\n", cmd.Name, cmd.Description)
					}
				}
				return b.String(), nil
			},
		},
		{
			Name:        "model",
			Category:    "Mode",
			Description: "Show or change the current model",
			Execute: func(ctx *CommandContext, args string) (string, error) {
				if args == "" {
					return fmt.Sprintf("Current model: %s", ctx.Model), nil
				}
				if ctx.SetModel == nil {
					return "Model switch not available.", nil
				}
				ctx.SetModel(args)
				return fmt.Sprintf("Model set to: %s", args), nil
			},
		},
		{
			Name:        "status",
			Aliases:     []string{"s"},
			Category:    "Info",
			Description: "Show session status",
			Execute: func(ctx *CommandContext, _ string) (string, error) {
				return fmt.Sprintf(
					"Model:    %s\nMode:     %s\nMessages: %d\nTokens:   %d\nCost:     $%.2f",
					ctx.Model, ctx.Mode, ctx.Messages, ctx.TotalTokens, ctx.TotalCost,
				), nil
			},
		},
		{
			Name:        "init",
			Category:    "Info",
			Description: "Initialize project configuration",
			Execute: func(_ *CommandContext, _ string) (string, error) {
				return "Project initialized.", nil
			},
		},
		{
			Name:        "memory",
			Category:    "Info",
			Description: "Show memory entries",
			Execute: func(ctx *CommandContext, _ string) (string, error) {
				if len(ctx.MemoryEntries) == 0 {
					return "No memory entries.", nil
				}
				var b strings.Builder
				b.WriteString("Memory entries:\n")
				for _, entry := range ctx.MemoryEntries {
					fmt.Fprintf(&b, "  - %s\n", entry)
				}
				return b.String(), nil
			},
		},
		{
			Name:        "context",
			Category:    "Info",
			Description: "Show current context info",
			Execute: func(ctx *CommandContext, _ string) (string, error) {
				return fmt.Sprintf(
					"CWD:   %s\nModel: %s\nMessages: %d",
					ctx.CWD, ctx.Model, ctx.Messages,
				), nil
			},
		},
		{
			Name:        "cost",
			Category:    "Info",
			Description: "Show session cost breakdown",
			Execute: func(ctx *CommandContext, _ string) (string, error) {
				return fmt.Sprintf(
					"Session cost: $%.4f\nTotal tokens: %d",
					ctx.TotalCost, ctx.TotalTokens,
				), nil
			},
		},
		{
			Name:        "exit",
			Category:    "Info",
			Description: "Exit the application",
			Execute: func(ctx *CommandContext, _ string) (string, error) {
				if ctx.ExitFn == nil {
					return "Exit not available.", nil
				}
				ctx.ExitFn()
				return "Goodbye.", nil
			},
		},
		{
			Name:        "plan",
			Category:    "Mode",
			Description: "Toggle plan mode",
			Execute: func(ctx *CommandContext, _ string) (string, error) {
				if ctx.ToggleMode == nil || ctx.GetMode == nil {
					return "Plan mode not available.", nil
				}
				ctx.ToggleMode()
				return fmt.Sprintf("Switched to %s mode.", ctx.GetMode()), nil
			},
		},
		{
			Name:        "rename",
			Category:    "Session",
			Description: "Rename current session",
			Execute: func(ctx *CommandContext, args string) (string, error) {
				if ctx.RenameSession == nil {
					return "Rename not available.", nil
				}
				if args == "" {
					return "Usage: /rename <name>", nil
				}
				ctx.RenameSession(args)
				return fmt.Sprintf("Session renamed to %q.", args), nil
			},
		},
		{
			Name:        "resume",
			Category:    "Session",
			Description: "Resume a previous session",
			Execute: func(ctx *CommandContext, args string) (string, error) {
				if args == "" {
					if ctx.ListSessionsFn != nil {
						return ctx.ListSessionsFn(), nil
					}
					return "Usage: /resume <id>", nil
				}
				if ctx.ResumeSession == nil {
					return "Resume not available.", nil
				}
				if err := ctx.ResumeSession(args); err != nil {
					return "", fmt.Errorf("resume session: %w", err)
				}
				return fmt.Sprintf("Resumed session %s.", args), nil
			},
		},
		{
			Name:        "sandbox",
			Category:    "Config",
			Description: "Show sandbox status",
			Execute: func(ctx *CommandContext, _ string) (string, error) {
				if ctx.SandboxStatus == nil {
					return "Sandbox status not available.", nil
				}
				return ctx.SandboxStatus(), nil
			},
		},
		{
			Name:        "vim",
			Category:    "Mode",
			Description: "Toggle vim mode",
			Execute: func(ctx *CommandContext, _ string) (string, error) {
				if ctx.ToggleVim == nil || ctx.VimEnabled == nil {
					return "Vim mode not available.", nil
				}
				ctx.ToggleVim()
				state := "disabled"
				if ctx.VimEnabled() {
					state = "enabled"
				}
				return fmt.Sprintf("Vim mode: %s.", state), nil
			},
		},
		{
			Name:        "mcp",
			Category:    "Config",
			Description: "List MCP servers",
			Execute: func(ctx *CommandContext, _ string) (string, error) {
				if ctx.MCPServers == nil {
					return "MCP servers not available.", nil
				}
				servers := ctx.MCPServers()
				if len(servers) == 0 {
					return "No MCP servers configured.", nil
				}
				var b strings.Builder
				b.WriteString("MCP servers:\n")
				for _, s := range servers {
					fmt.Fprintf(&b, "  - %s\n", s)
				}
				return b.String(), nil
			},
		},
		{
			Name:        "export",
			Category:    "Session",
			Description: "Export conversation to file (.md or .html)",
			Execute: func(ctx *CommandContext, args string) (string, error) {
				if ctx.ExportConversation == nil {
					return "Export not available.", nil
				}
				if args == "" {
					return "Usage: /export <path>", nil
				}
				ext := strings.ToLower(filepath.Ext(args))
				if ext == ".html" || ext == ".htm" {
					if ctx.ExportHTMLFn != nil {
						if err := ctx.ExportHTMLFn(args); err != nil {
							return "", fmt.Errorf("export HTML: %w", err)
						}
						return fmt.Sprintf("Exported HTML to %s.", args), nil
					}
					return "HTML export not available.", nil
				}
				if err := ctx.ExportConversation(args); err != nil {
					return "", fmt.Errorf("export conversation: %w", err)
				}
				return fmt.Sprintf("Exported to %s.", args), nil
			},
		},
		{
			Name:        "tree",
			Category:    "Session",
			Description: "Show session tree (branch structure)",
			Execute: func(ctx *CommandContext, _ string) (string, error) {
				if ctx.SessionTreeFn != nil {
					return ctx.SessionTreeFn(), nil
				}
				return "Session tree not available.", nil
			},
		},
		{
			Name:        "scoped-models",
			Category:    "Mode",
			Description: "Manage scoped models configuration",
			Execute: func(ctx *CommandContext, _ string) (string, error) {
				if ctx.ScopedModelsFn != nil {
					return ctx.ScopedModelsFn(), nil
				}
				return "Scoped models not available.", nil
			},
		},
		{
			Name:        "reload",
			Category:    "Config",
			Description: "Reload configuration files",
			Execute: func(ctx *CommandContext, _ string) (string, error) {
				if ctx.ReloadFn == nil {
					return "Reload not available.", nil
				}
				return ctx.ReloadFn()
			},
		},
		{
			Name:        "hooks",
			Category:    "Config",
			Description: "Show and manage hooks",
			Execute: func(ctx *CommandContext, _ string) (string, error) {
				if ctx.HookManagerFn != nil {
					return ctx.HookManagerFn(), nil
				}
				return "Hook manager not available.", nil
			},
		},
		{
			Name:        "permissions",
			Category:    "Config",
			Description: "Show and manage permission rules",
			Execute: func(ctx *CommandContext, _ string) (string, error) {
				if ctx.PermissionManagerFn != nil {
					return ctx.PermissionManagerFn(), nil
				}
				return "Permission manager not available.", nil
			},
		},
		{
			Name:        "hotkeys",
			Category:    "Info",
			Description: "Show keyboard shortcuts",
			Execute: func(ctx *CommandContext, _ string) (string, error) {
				if ctx.KeybindingsFn != nil {
					return ctx.KeybindingsFn(), nil
				}
				return defaultHotkeysTable(), nil
			},
		},
		{
			Name:        "changelog",
			Category:    "Info",
			Description: "Show version history",
			Execute: func(_ *CommandContext, _ string) (string, error) {
				return changelog.Get(), nil
			},
		},
		{
			Name:        "settings",
			Category:    "Config",
			Description: "Show current settings",
			Execute: func(ctx *CommandContext, _ string) (string, error) {
				if ctx.GetSettings != nil {
					return ctx.GetSettings(), nil
				}
				return "Settings not available.", nil
			},
		},
		{
			Name:        "share",
			Category:    "Session",
			Description: "Share current session",
			Execute: func(ctx *CommandContext, _ string) (string, error) {
				if ctx.ShareFn != nil {
					return ctx.ShareFn(), nil
				}
				return "Share not available.", nil
			},
		},
		{
			Name:        "copy",
			Category:    "Session",
			Description: "Copy last assistant message to clipboard",
			Execute: func(ctx *CommandContext, _ string) (string, error) {
				if ctx.CopyLastMessageFn == nil {
					return "Copy not available.", nil
				}
				return ctx.CopyLastMessageFn()
			},
		},
		{
			Name:        "new",
			Category:    "Session",
			Description: "Start a new session",
			Execute: func(ctx *CommandContext, _ string) (string, error) {
				if ctx.NewSessionFn == nil {
					return "New session not available.", nil
				}
				ctx.NewSessionFn()
				return "Started new session.", nil
			},
		},
		{
			Name:        "fork",
			Category:    "Session",
			Description: "Fork current session",
			Execute: func(ctx *CommandContext, _ string) (string, error) {
				if ctx.ForkSessionFn == nil {
					return "Fork not available.", nil
				}
				newID, err := ctx.ForkSessionFn()
				if err != nil {
					return "", fmt.Errorf("fork session: %w", err)
				}
				return fmt.Sprintf("Forked session: %s", newID), nil
			},
		},
		{
			Name:        "quit",
			Aliases:     []string{"q"},
			Category:    "Info",
			Description: "Exit the application (alias for /exit)",
			Execute: func(ctx *CommandContext, _ string) (string, error) {
				if ctx.ExitFn == nil {
					return "Exit not available.", nil
				}
				ctx.ExitFn()
				return "Goodbye.", nil
			},
		},
	}
	for _, cmd := range core {
		r.commands[cmd.Name] = cmd
	}
}
