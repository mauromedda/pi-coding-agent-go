// ABOUTME: Directory listing tool: lists entries with name, size, and mod time
// ABOUTME: Read-only tool that wraps os.ReadDir with formatted output

package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"github.com/mauromedda/pi-coding-agent-go/internal/agent"
)

// NewLsTool creates a read-only tool that lists directory contents.
func NewLsTool() *agent.AgentTool {
	return &agent.AgentTool{
		Name:        "ls",
		Label:       "List Directory",
		Description: "List the contents of a directory with name, size, and modification time.",
		Parameters: json.RawMessage(`{
			"type": "object",
			"required": ["path"],
			"properties": {
				"path": {"type": "string", "description": "Absolute path to the directory"}
			}
		}`),
		ReadOnly: true,
		Execute:  executeLs,
	}
}

func executeLs(_ context.Context, _ string, params map[string]any, _ func(agent.ToolUpdate)) (agent.ToolResult, error) {
	path, err := requireStringParam(params, "path")
	if err != nil {
		return errResult(err), nil
	}

	entries, err := os.ReadDir(path)
	if err != nil {
		return errResult(fmt.Errorf("reading directory %s: %w", path, err)), nil
	}

	output := formatEntries(entries)
	return agent.ToolResult{Content: output}, nil
}

// formatEntries formats directory entries as a human-readable listing.
func formatEntries(entries []os.DirEntry) string {
	if len(entries) == 0 {
		return "(empty directory)"
	}

	var b strings.Builder
	for _, e := range entries {
		info, err := e.Info()
		if err != nil {
			fmt.Fprintf(&b, "%s  (info unavailable)\n", e.Name())
			continue
		}

		size := info.Size()
		modTime := info.ModTime().Format("2006-01-02 15:04:05")
		prefix := " "
		if e.IsDir() {
			prefix = "d"
		}

		fmt.Fprintf(&b, "%s %10d  %s  %s\n", prefix, size, modTime, e.Name())
	}
	return b.String()
}
