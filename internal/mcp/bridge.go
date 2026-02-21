// ABOUTME: Converts MCP tools into agent.AgentTool instances for the tool registry
// ABOUTME: Names tools as mcp__<server>__<tool> following Claude Code convention

package mcp

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/mauromedda/pi-coding-agent-go/internal/agent"
)

// BridgeTool converts an MCPTool from a named server into an AgentTool.
func BridgeTool(serverName string, tool MCPTool, client *Client) *agent.AgentTool {
	name := fmt.Sprintf("mcp__%s__%s", sanitizeName(serverName), sanitizeName(tool.Name))

	return &agent.AgentTool{
		Name:        name,
		Label:       tool.Name,
		Description: tool.Description,
		Parameters:  tool.InputSchema,
		ReadOnly:    false,
		Execute: func(ctx context.Context, id string, params map[string]any, onUpdate func(agent.ToolUpdate)) (agent.ToolResult, error) {
			result, err := client.CallTool(ctx, tool.Name, params)
			if err != nil {
				return agent.ToolResult{Content: err.Error(), IsError: true}, nil
			}

			var text strings.Builder
			for _, item := range result.Content {
				if item.Text != "" {
					if text.Len() > 0 {
						text.WriteString("\n")
					}
					text.WriteString(item.Text)
				}
			}

			return agent.ToolResult{
				Content: text.String(),
				IsError: result.IsError,
			}, nil
		},
	}
}

// BridgeAllTools converts all tools from a client into AgentTools.
func BridgeAllTools(serverName string, client *Client) []*agent.AgentTool {
	tools := client.Tools()
	result := make([]*agent.AgentTool, len(tools))
	for i, tool := range tools {
		result[i] = BridgeTool(serverName, tool, client)
	}
	return result
}

// sanitizeName replaces characters not safe for tool names.
func sanitizeName(s string) string {
	return strings.Map(func(r rune) rune {
		if (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9') || r == '_' {
			return r
		}
		return '_'
	}, s)
}

// ServerToolJSON is used for tool discovery without a live client.
type ServerToolJSON struct {
	Name        string          `json:"name"`
	Description string          `json:"description,omitempty"`
	InputSchema json.RawMessage `json:"inputSchema,omitempty"`
	ServerName  string          `json:"serverName"`
}
