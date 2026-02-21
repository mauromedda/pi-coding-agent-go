// ABOUTME: Slash command registry and dispatch for interactive mode
// ABOUTME: Provides /clear, /compact, /config, /help, /model, /status commands

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
	}
	for _, cmd := range core {
		r.commands[cmd.Name] = cmd
	}
}
