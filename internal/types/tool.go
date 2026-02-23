// ABOUTME: Shared tool types decoupled from the agent package
// ABOUTME: Breaks the agent → mcp circular dependency via a common types package

package types

import (
	"context"
	"encoding/json"
	"time"
)

// ImageBlock carries image data through the tool result pipeline.
// Not serialized to JSON; used only for in-process rendering.
type ImageBlock struct {
	Data     []byte // Raw image bytes
	MimeType string // e.g. "image/png"
	Filename string
}

// ToolResult holds the outcome of a single tool execution.
type ToolResult struct {
	Content  string
	IsError  bool
	Duration time.Duration
	Images   []ImageBlock `json:"-"` // In-process only; not serialized
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
