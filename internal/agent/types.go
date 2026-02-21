// ABOUTME: Core agent types: events, tool definitions, and state enumerations
// ABOUTME: Wire-format agnostic; re-exports shared types from internal/types

package agent

import (
	"github.com/mauromedda/pi-coding-agent-go/internal/types"
)

// Re-export shared types so existing consumers of internal/agent continue to compile.
type AgentTool = types.AgentTool
type ToolResult = types.ToolResult
type ToolUpdate = types.ToolUpdate

// AgentEventType identifies the kind of agent event emitted during execution.
type AgentEventType int

const (
	EventAgentStart       AgentEventType = iota // Agent loop started
	EventAgentEnd                               // Agent loop finished
	EventAssistantText                          // Streamed text from the model
	EventAssistantThinking                      // Extended thinking output
	EventToolStart                              // Tool execution began
	EventToolUpdate                             // Incremental tool output
	EventToolEnd                                // Tool execution completed
	EventError                                  // Non-recoverable error
)

// AgentEvent represents a single event emitted by the agent loop.
type AgentEvent struct {
	Type       AgentEventType
	Text       string
	ToolID     string
	ToolName   string
	ToolArgs   map[string]any
	ToolResult *ToolResult
	Error      error
}

// AgentState represents the current lifecycle state of the agent.
type AgentState int32

const (
	StateIdle      AgentState = iota // Not running
	StateRunning                     // Actively processing
	StateCancelled                   // Cancelled by caller
)
