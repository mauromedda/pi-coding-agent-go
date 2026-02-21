// ABOUTME: Read-file tool: returns file contents with optional offset/limit
// ABOUTME: Detects binary files and truncates large outputs (>100KB)

package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"github.com/mauromedda/pi-coding-agent-go/internal/agent"
)

const (
	maxReadOutput    = 100 * 1024 // 100KB
	binaryCheckBytes = 512
)

// NewReadTool creates a read-only tool that returns file contents.
func NewReadTool() *agent.AgentTool {
	return &agent.AgentTool{
		Name:        "read",
		Label:       "Read File",
		Description: "Read the contents of a file. Supports optional offset and limit in lines.",
		Parameters: json.RawMessage(`{
			"type": "object",
			"required": ["path"],
			"properties": {
				"path":   {"type": "string", "description": "Absolute path to the file"},
				"offset": {"type": "integer", "description": "Line number to start reading from (0-based)"},
				"limit":  {"type": "integer", "description": "Maximum number of lines to return"}
			}
		}`),
		ReadOnly: true,
		Execute:  executeRead,
	}
}

func executeRead(_ context.Context, _ string, params map[string]any, _ func(agent.ToolUpdate)) (agent.ToolResult, error) {
	path, err := requireStringParam(params, "path")
	if err != nil {
		return errResult(err), nil
	}

	data, err := os.ReadFile(path)
	if err != nil {
		return errResult(fmt.Errorf("reading file %s: %w", path, err)), nil
	}

	if isBinary(data) {
		return agent.ToolResult{Content: fmt.Sprintf("binary file detected: %s", path), IsError: true}, nil
	}

	content := applyOffsetLimit(string(data), params)
	content = truncateOutput(content, maxReadOutput)

	return agent.ToolResult{Content: content}, nil
}

// isBinary checks for null bytes in the first binaryCheckBytes of data.
func isBinary(data []byte) bool {
	limit := len(data)
	if limit > binaryCheckBytes {
		limit = binaryCheckBytes
	}
	for _, b := range data[:limit] {
		if b == 0 {
			return true
		}
	}
	return false
}

// applyOffsetLimit extracts a line range from content based on offset/limit params.
func applyOffsetLimit(content string, params map[string]any) string {
	lines := splitLines(content)

	offset := intParam(params, "offset", 0)
	limit := intParam(params, "limit", 0)

	if offset > len(lines) {
		offset = len(lines)
	}
	lines = lines[offset:]

	if limit > 0 && limit < len(lines) {
		lines = lines[:limit]
	}

	return joinLines(lines)
}

// splitLines splits content into lines, preserving trailing newlines.
func splitLines(s string) []string {
	if s == "" {
		return nil
	}
	return strings.SplitAfter(s, "\n")
}

// joinLines concatenates lines back into a single string using strings.Builder.
func joinLines(lines []string) string {
	var b strings.Builder
	for _, l := range lines {
		b.WriteString(l)
	}
	return b.String()
}

// truncateOutput limits output to maxBytes, appending a truncation notice.
func truncateOutput(s string, maxBytes int) string {
	if len(s) <= maxBytes {
		return s
	}
	return s[:maxBytes] + "\n... [output truncated]"
}
