// ABOUTME: Task tool allowing the LLM to spawn sub-agents by name
// ABOUTME: Wraps agent.Spawn with agent definition lookup and result formatting

package tools

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/mauromedda/pi-coding-agent-go/internal/agent"
)

// NewTaskTool creates a tool that spawns sub-agents.
func NewTaskTool(deps agent.SpawnDeps, defs map[string]agent.Definition) *agent.AgentTool {
	return &agent.AgentTool{
		Name:        "task",
		Label:       "Launch Sub-Agent",
		Description: "Launch a specialized agent to handle a task. Available agents: explore, plan, bash_agent, or custom-defined agents.",
		Parameters: json.RawMessage(`{
			"type": "object",
			"required": ["agent", "prompt"],
			"properties": {
				"agent":      {"type": "string", "description": "Agent name to spawn (e.g., explore, plan)"},
				"prompt":     {"type": "string", "description": "Task description for the agent"},
				"background": {"type": "boolean", "description": "Run in background (default: false)"}
			}
		}`),
		ReadOnly: true,
		Execute: func(ctx context.Context, id string, params map[string]any, onUpdate func(agent.ToolUpdate)) (agent.ToolResult, error) {
			agentName, err := requireStringParam(params, "agent")
			if err != nil {
				return errResult(err), nil
			}

			prompt, err := requireStringParam(params, "prompt")
			if err != nil {
				return errResult(err), nil
			}

			background := boolParam(params, "background", false)

			def, ok := defs[agentName]
			if !ok {
				return errResult(fmt.Errorf("unknown agent: %q", agentName)), nil
			}

			cfg := agent.SubAgentConfig{
				Name:            def.Name,
				Description:     def.Description,
				Model:           def.Model,
				SystemPrompt:    def.SystemPrompt,
				Tools:           def.Tools,
				DisallowedTools: def.DisallowedTools,
				MaxTurns:        def.MaxTurns,
				Background:      background,
			}

			handle, err := agent.Spawn(ctx, cfg, prompt, deps)
			if err != nil {
				return errResult(fmt.Errorf("spawning agent %q: %w", agentName, err)), nil
			}

			if background {
				return agent.ToolResult{
					Content: fmt.Sprintf("Agent %q spawned in background (id: %s)", agentName, handle.ID),
				}, nil
			}

			// Foreground: wait for result
			<-handle.Done
			result := handle.Result()
			if result == nil {
				return agent.ToolResult{Content: "agent completed with no result"}, nil
			}
			if result.Error != nil {
				return agent.ToolResult{Content: result.Text + "\nError: " + result.Error.Error(), IsError: true}, nil
			}
			return agent.ToolResult{Content: result.Text}, nil
		},
	}
}
