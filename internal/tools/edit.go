// ABOUTME: Edit tool: surgical text replacement within existing files
// ABOUTME: Validates paths via sandbox; supports single and replace-all modes

package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"github.com/mauromedda/pi-coding-agent-go/internal/agent"
	"github.com/mauromedda/pi-coding-agent-go/internal/diff"
	"github.com/mauromedda/pi-coding-agent-go/internal/permission"
)

// NewEditTool creates a tool that performs text replacement in files.
func NewEditTool() *agent.AgentTool {
	return newEditTool(nil)
}

// NewEditToolWithSandbox creates an edit tool that validates paths against the sandbox.
func NewEditToolWithSandbox(sb *permission.Sandbox) *agent.AgentTool {
	return newEditTool(sb)
}

func newEditTool(sb *permission.Sandbox) *agent.AgentTool {
	return &agent.AgentTool{
		Name:        "edit",
		Label:       "Edit File",
		Description: `Performs exact string replacements in files.

Usage:
- The edit will FAIL if old_string is not found in the file
- The edit will FAIL if old_string is not unique in the file (appears more than once)
  unless replace_all is set to true
- Exact whitespace matching is required: old_string must match the file content exactly,
  including indentation (tabs vs spaces) and line endings
- Use replace_all for renaming variables or replacing repeated patterns across the file

Output:
- Returns a unified diff showing the changes made
- Maximum file size: 10MB

Parameters:
- path (required): Absolute path to the file to modify
- old_string (required): The exact text to find in the file
- new_string (required): The replacement text (must differ from old_string)
- replace_all: Set to true to replace all occurrences (default: false)`,
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
		Execute: func(ctx context.Context, id string, params map[string]any, onUpdate func(agent.ToolUpdate)) (agent.ToolResult, error) {
			return executeEdit(sb, ctx, id, params, onUpdate)
		},
	}
}

func executeEdit(sb *permission.Sandbox, _ context.Context, _ string, params map[string]any, _ func(agent.ToolUpdate)) (agent.ToolResult, error) {
	rawPath, err := requireStringParam(params, "path")
	if err != nil {
		return errResult(err), nil
	}

	path := ExpandPath(rawPath)

	if sb != nil {
		if err := sb.ValidatePath(path); err != nil {
			return errResult(err), nil
		}
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

	info, err := os.Stat(path)
	if err != nil {
		return errResult(fmt.Errorf("stat file %s: %w", path, err)), nil
	}
	if info.Size() > maxFileReadSize {
		return errResult(fmt.Errorf("file %s is too large (%d bytes); maximum is %d bytes", path, info.Size(), maxFileReadSize)), nil
	}

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

	d := diff.Simple(path, original, result)
	return agent.ToolResult{Content: d}, nil
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
