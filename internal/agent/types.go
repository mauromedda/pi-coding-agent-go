// ABOUTME: Core agent types: events, tool definitions, and state enumerations
// ABOUTME: Wire-format agnostic; used by agent loop and tool implementations

package agent

import (
	"context"
	"encoding/json"
	"time"
)

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

// ToolResult holds the outcome of a single tool execution.
type ToolResult struct {
	Content  string
	IsError  bool
	Duration time.Duration
}

// ToolUpdate carries incremental output from a running tool.
type ToolUpdate struct {
	Output string
}

// AgentTool defines a tool that the agent can invoke during its loop.
type AgentTool struct {
	Name        string
	Label       string
	Description string
	Parameters  json.RawMessage
	ReadOnly    bool
	Execute     func(ctx context.Context, id string, params map[string]any, onUpdate func(ToolUpdate)) (ToolResult, error)
}

// AgentState represents the current lifecycle state of the agent.
type AgentState int32

const (
	StateIdle      AgentState = iota // Not running
	StateRunning                     // Actively processing
	StateCancelled                   // Cancelled by caller
)
