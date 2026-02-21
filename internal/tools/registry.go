// ABOUTME: Tool registry: creates, stores, and queries agent tools
// ABOUTME: Auto-detects ripgrep availability and registers all built-in tools

package tools

import (
	"os/exec"

	"github.com/mauromedda/pi-coding-agent-go/internal/agent"
)

// Registry manages the collection of available agent tools.
type Registry struct {
	tools map[string]*agent.AgentTool
	hasRg bool
}

// NewRegistry creates a Registry, auto-detects ripgrep, and registers built-in tools.
func NewRegistry() *Registry {
	r := &Registry{
		tools: make(map[string]*agent.AgentTool),
		hasRg: detectRipgrep(),
	}
	r.registerBuiltins()
	return r
}

// Register adds a tool to the registry, replacing any existing tool with the same name.
func (r *Registry) Register(tool *agent.AgentTool) {
	r.tools[tool.Name] = tool
}

// Get returns a tool by name, or nil if not found.
func (r *Registry) Get(name string) *agent.AgentTool {
	return r.tools[name]
}

// All returns every registered tool as a slice.
func (r *Registry) All() []*agent.AgentTool {
	out := make([]*agent.AgentTool, 0, len(r.tools))
	for _, t := range r.tools {
		out = append(out, t)
	}
	return out
}

// ReadOnly returns only tools whose ReadOnly flag is true.
func (r *Registry) ReadOnly() []*agent.AgentTool {
	var out []*agent.AgentTool
	for _, t := range r.tools {
		if t.ReadOnly {
			out = append(out, t)
		}
	}
	return out
}

// HasRipgrep reports whether ripgrep (rg) was found on PATH.
func (r *Registry) HasRipgrep() bool {
	return r.hasRg
}

// registerBuiltins adds all built-in tools to the registry.
func (r *Registry) registerBuiltins() {
	builtins := []*agent.AgentTool{
		NewReadTool(),
		NewWriteTool(),
		NewEditTool(),
		NewBashTool(),
		NewGrepTool(r.hasRg),
		NewFindTool(r.hasRg),
		NewLsTool(),
	}
	for _, t := range builtins {
		r.Register(t)
	}
}

// detectRipgrep checks whether rg is available on PATH.
func detectRipgrep() bool {
	_, err := exec.LookPath("rg")
	return err == nil
}
