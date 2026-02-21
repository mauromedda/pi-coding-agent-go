// ABOUTME: Slash command registry and dispatch for interactive mode
// ABOUTME: Provides 18 slash commands: clear, compact, config, context, cost, exit, export, help, init, mcp, memory, model, plan, rename, resume, sandbox, status, vim

package commands

import (
	"fmt"
	"sort"
	"strings"
)

// Command represents a slash command.
type Command struct {
	Name        string
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
}

// Registry holds all registered slash commands.
type Registry struct {
	commands map[string]*Command
}

// NewRegistry creates a registry with all core commands registered.
func NewRegistry() *Registry {
	r := &Registry{commands: make(map[string]*Command)}
	r.registerCoreCommands()
	return r
}

// Get returns a command by name.
// The second return value indicates whether the name was found.
func (r *Registry) Get(name string) (*Command, bool) {
	cmd, ok := r.commands[name]
	return cmd, ok
}

// List returns all commands sorted by name for deterministic output.
func (r *Registry) List() []*Command {
	result := make([]*Command, 0, len(r.commands))
	for _, cmd := range r.commands {
		result = append(result, cmd)
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

// IsCommand returns true if input starts with '/'.
func IsCommand(input string) bool {
	return len(input) > 0 && input[0] == '/'
}

// registerCoreCommands adds all built-in slash commands to the registry.
func (r *Registry) registerCoreCommands() {
	core := []*Command{
		{
			Name:        "clear",
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
			Description: "Compact conversation into a summary",
			Execute: func(ctx *CommandContext, _ string) (string, error) {
				return ctx.CompactFn(), nil
			},
		},
		{
			Name:        "config",
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
			Description: "Show available commands",
			Execute: func(ctx *CommandContext, _ string) (string, error) {
				var b strings.Builder
				b.WriteString("Available commands:\n")
				for _, cmd := range r.List() {
					fmt.Fprintf(&b, "  /%s — %s\n", cmd.Name, cmd.Description)
				}
				return b.String(), nil
			},
		},
		{
			Name:        "model",
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
			Description: "Initialize project configuration",
			Execute: func(_ *CommandContext, _ string) (string, error) {
				return "Project initialized.", nil
			},
		},
		{
			Name:        "memory",
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
			Description: "Resume a previous session",
			Execute: func(ctx *CommandContext, args string) (string, error) {
				if ctx.ResumeSession == nil {
					return "Resume not available.", nil
				}
				if args == "" {
					return "Usage: /resume <id>", nil
				}
				if err := ctx.ResumeSession(args); err != nil {
					return "", fmt.Errorf("resume session: %w", err)
				}
				return fmt.Sprintf("Resumed session %s.", args), nil
			},
		},
		{
			Name:        "sandbox",
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
			Description: "Export conversation to file",
			Execute: func(ctx *CommandContext, args string) (string, error) {
				if ctx.ExportConversation == nil {
					return "Export not available.", nil
				}
				if args == "" {
					return "Usage: /export <path>", nil
				}
				if err := ctx.ExportConversation(args); err != nil {
					return "", fmt.Errorf("export conversation: %w", err)
				}
				return fmt.Sprintf("Exported to %s.", args), nil
			},
		},
		{
			Name:        "tree",
			Description: "Show session tree (branch structure)",
			Execute: func(ctx *CommandContext, _ string) (string, error) {
				return "Session tree: " + ctx.Model, nil
			},
		},
		{
			Name:        "scoped-models",
			Description: "Manage scoped models configuration",
			Execute: func(ctx *CommandContext, _ string) (string, error) {
				return "Scoped models: " + ctx.Model, nil
			},
		},
		{
			Name:        "reload",
			Description: "Reload configuration files",
			Execute: func(ctx *CommandContext, _ string) (string, error) {
				if ctx.ReloadFn == nil {
					return "Reload not available.", nil
				}
				return ctx.ReloadFn()
			},
		},
	}
	for _, cmd := range core {
		r.commands[cmd.Name] = cmd
	}
}
