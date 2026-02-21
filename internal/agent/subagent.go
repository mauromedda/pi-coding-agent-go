// ABOUTME: Sub-agent spawning with isolated context, tool filtering, and max turns
// ABOUTME: Supports foreground and background execution with result collection

package agent

import (
	"context"
	"fmt"
	"strings"
	"sync/atomic"

	"github.com/mauromedda/pi-coding-agent-go/pkg/ai"
)

// SubAgentConfig describes how to spawn a sub-agent.
type SubAgentConfig struct {
	Name            string
	Description     string
	Model           string
	SystemPrompt    string
	Tools           []string // Allowlist (nil = inherit all)
	DisallowedTools []string // Blocklist
	MaxTurns        int
	Background      bool
}

// SubAgentResult holds the outcome of a sub-agent execution.
type SubAgentResult struct {
	Text  string
	Error error
}

// SubAgentHandle tracks a running sub-agent.
type SubAgentHandle struct {
	ID     string
	Name   string
	Done   <-chan struct{}
	result atomic.Pointer[SubAgentResult]
}

// Result returns the sub-agent result, or nil if still running.
func (h *SubAgentHandle) Result() *SubAgentResult {
	return h.result.Load()
}

// SpawnDeps provides dependencies for spawning sub-agents.
type SpawnDeps struct {
	Provider ai.ApiProvider
	Model    *ai.Model
	AllTools []*AgentTool
}

// Spawn creates and runs a sub-agent with isolated context.
func Spawn(ctx context.Context, cfg SubAgentConfig, prompt string, deps SpawnDeps) (*SubAgentHandle, error) {
	tools := filterTools(deps.AllTools, cfg.Tools, cfg.DisallowedTools)

	system := cfg.SystemPrompt
	if system == "" {
		system = fmt.Sprintf("You are %s. %s", cfg.Name, cfg.Description)
	}

	llmCtx := &ai.Context{
		System: system,
		Messages: []ai.Message{
			{Role: ai.RoleUser, Content: []ai.Content{{Type: ai.ContentText, Text: prompt}}},
		},
		Tools: aiTools(toolMap(tools)),
	}

	opts := &ai.StreamOptions{
		MaxTokens: 4096,
	}

	ag := New(deps.Provider, deps.Model, tools)

	done := make(chan struct{})
	handle := &SubAgentHandle{
		ID:   generateID(),
		Name: cfg.Name,
		Done: done,
	}

	run := func() {
		defer close(done)
		result := runSubAgent(ctx, ag, llmCtx, opts, cfg.MaxTurns)
		handle.result.Store(result)
	}

	if cfg.Background {
		go run()
	} else {
		run()
	}

	return handle, nil
}

// runSubAgent executes the agent loop with turn limiting and collects text output.
func runSubAgent(ctx context.Context, ag *Agent, llmCtx *ai.Context, opts *ai.StreamOptions, maxTurns int) *SubAgentResult {
	if maxTurns <= 0 {
		maxTurns = 10 // Default
	}

	var text strings.Builder
	turns := 0

	for turns < maxTurns {
		events := ag.Prompt(ctx, llmCtx, opts)
		turns++

		hasToolUse := false
		for evt := range events {
			switch evt.Type {
			case EventAssistantText:
				text.WriteString(evt.Text)
			case EventToolEnd:
				hasToolUse = true
			case EventError:
				return &SubAgentResult{Text: text.String(), Error: evt.Error}
			}
		}

		if !hasToolUse {
			break
		}
	}

	return &SubAgentResult{Text: text.String()}
}

// filterTools applies allow/disallow lists to produce a tool subset.
func filterTools(all []*AgentTool, allow, disallow []string) []*AgentTool {
	if len(allow) == 0 && len(disallow) == 0 {
		return all
	}

	disallowSet := make(map[string]bool, len(disallow))
	for _, name := range disallow {
		disallowSet[name] = true
	}

	if len(allow) > 0 {
		allowSet := make(map[string]bool, len(allow))
		for _, name := range allow {
			allowSet[name] = true
		}

		var result []*AgentTool
		for _, t := range all {
			if allowSet[t.Name] && !disallowSet[t.Name] {
				result = append(result, t)
			}
		}
		return result
	}

	// Only disallow list
	var result []*AgentTool
	for _, t := range all {
		if !disallowSet[t.Name] {
			result = append(result, t)
		}
	}
	return result
}

// toolMap converts a slice to a map.
func toolMap(tools []*AgentTool) map[string]*AgentTool {
	m := make(map[string]*AgentTool, len(tools))
	for _, t := range tools {
		m[t.Name] = t
	}
	return m
}

var idCounter atomic.Int64

func generateID() string {
	return fmt.Sprintf("sub-%d", idCounter.Add(1))
}
