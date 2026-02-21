// ABOUTME: Edit tool: surgical text replacement within existing files
// ABOUTME: Supports single and replace-all modes; generates unified diff output

package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"github.com/mauromedda/pi-coding-agent-go/internal/agent"
)

// NewEditTool creates a tool that performs text replacement in files.
func NewEditTool() *agent.AgentTool {
	return &agent.AgentTool{
		Name:        "edit",
		Label:       "Edit File",
		Description: "Replace occurrences of old_string with new_string in a file.",
		Parameters: json.RawMessage(`{
			"type": "object",
			"required": ["path", "old_string", "new_string"],
			"properties": {
				"path":        {"type": "string", "description": "Absolute path to the file"},
				"old_string":  {"type": "string", "description": "Text to find"},
				"new_string":  {"type": "string", "description": "Replacement text"},
				"replace_all": {"type": "boolean", "description": "Replace all occurrences (default false)"}
			}
		}`),
		ReadOnly: false,
		Execute:  executeEdit,
	}
}

func executeEdit(_ context.Context, _ string, params map[string]any, _ func(agent.ToolUpdate)) (agent.ToolResult, error) {
	path, err := requireStringParam(params, "path")
	if err != nil {
		return errResult(err), nil
	}

	oldStr, err := requireStringParam(params, "old_string")
	if err != nil {
		return errResult(err), nil
	}

	newStr, err := requireStringParam(params, "new_string")
	if err != nil {
		return errResult(err), nil
	}

	replaceAll := boolParam(params, "replace_all", false)

	data, err := os.ReadFile(path)
	if err != nil {
		return errResult(fmt.Errorf("reading file %s: %w", path, err)), nil
	}

	original := string(data)
	result, err := applyReplacement(original, oldStr, newStr, replaceAll)
	if err != nil {
		return errResult(err), nil
	}

	if err := os.WriteFile(path, []byte(result), 0o644); err != nil {
		return errResult(fmt.Errorf("writing file %s: %w", path, err)), nil
	}

	diff := simpleDiff(path, original, result)
	return agent.ToolResult{Content: diff}, nil
}

// applyReplacement performs the string substitution with uniqueness checks.
func applyReplacement(content, oldStr, newStr string, replaceAll bool) (string, error) {
	count := strings.Count(content, oldStr)
	if count == 0 {
		return "", fmt.Errorf("old_string not found in file")
	}
	if count > 1 && !replaceAll {
		return "", fmt.Errorf("old_string found %d times; set replace_all=true to replace all", count)
	}

	if replaceAll {
		return strings.ReplaceAll(content, oldStr, newStr), nil
	}

	return strings.Replace(content, oldStr, newStr, 1), nil
}

// simpleDiff produces a minimal unified-style diff of the changes.
func simpleDiff(path, before, after string) string {
	oldLines := strings.Split(before, "\n")
	newLines := strings.Split(after, "\n")

	var b strings.Builder
	fmt.Fprintf(&b, "--- %s\n+++ %s\n", path, path)

	maxLen := len(oldLines)
	if len(newLines) > maxLen {
		maxLen = len(newLines)
	}

	for i := 0; i < maxLen; i++ {
		oldLine := lineAt(oldLines, i)
		newLine := lineAt(newLines, i)
		if oldLine != newLine {
			if i < len(oldLines) {
				fmt.Fprintf(&b, "-%s\n", oldLine)
			}
			if i < len(newLines) {
				fmt.Fprintf(&b, "+%s\n", newLine)
			}
		}
	}

	return b.String()
}

// lineAt safely returns the line at index i, or empty string if out of range.
func lineAt(lines []string, i int) string {
	if i < len(lines) {
		return lines[i]
	}
	return ""
}
