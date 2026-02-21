// ABOUTME: NotebookEdit tool for editing Jupyter .ipynb notebook cells
// ABOUTME: Supports replace, insert, and delete operations on notebook cells

package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"github.com/mauromedda/pi-coding-agent-go/internal/agent"
)

// notebook represents a Jupyter notebook file structure.
type notebook struct {
	Cells         []cell         `json:"cells"`
	Metadata      map[string]any `json:"metadata"`
	NBFormat      int            `json:"nbformat"`
	NBFormatMinor int            `json:"nbformat_minor"`
}

// cell represents a single cell in a Jupyter notebook.
type cell struct {
	CellType string         `json:"cell_type"`
	Source   []string       `json:"source"`
	Metadata map[string]any `json:"metadata"`
	Outputs  []any          `json:"outputs,omitempty"`
}

// NewNotebookEditTool creates a tool that edits Jupyter notebook cells.
func NewNotebookEditTool() *agent.AgentTool {
	return &agent.AgentTool{
		Name:        "notebook_edit",
		Label:       "Edit Notebook",
		Description: "Edit Jupyter notebook cells: replace, insert, or delete",
		Parameters: json.RawMessage(`{
			"type": "object",
			"properties": {
				"path":        {"type": "string", "description": "Path to .ipynb file"},
				"cell_number": {"type": "integer", "description": "0-based cell index"},
				"operation":   {"type": "string", "enum": ["replace", "insert", "delete"]},
				"cell_type":   {"type": "string", "enum": ["code", "markdown"]},
				"source":      {"type": "string", "description": "New cell content"}
			},
			"required": ["path", "cell_number", "operation"]
		}`),
		ReadOnly: false,
		Execute: func(ctx context.Context, id string, params map[string]any, onUpdate func(agent.ToolUpdate)) (agent.ToolResult, error) {
			return executeNotebookEdit(params)
		},
	}
}

func executeNotebookEdit(params map[string]any) (agent.ToolResult, error) {
	path, err := requireStringParam(params, "path")
	if err != nil {
		return errResult(err), nil
	}

	cellNum := intParam(params, "cell_number", -999)
	if cellNum == -999 {
		return errResult(fmt.Errorf("missing required parameter %q", "cell_number")), nil
	}

	op, err := requireStringParam(params, "operation")
	if err != nil {
		return errResult(err), nil
	}

	data, err := os.ReadFile(path)
	if err != nil {
		return errResult(fmt.Errorf("reading notebook %s: %w", path, err)), nil
	}

	var nb notebook
	if err := json.Unmarshal(data, &nb); err != nil {
		return errResult(fmt.Errorf("parsing notebook JSON: %w", err)), nil
	}

	switch op {
	case "replace":
		return notebookReplace(&nb, path, cellNum, params)
	case "insert":
		return notebookInsert(&nb, path, cellNum, params)
	case "delete":
		return notebookDelete(&nb, path, cellNum)
	default:
		return errResult(fmt.Errorf("unknown operation %q; expected replace, insert, or delete", op)), nil
	}
}

func notebookReplace(nb *notebook, path string, idx int, params map[string]any) (agent.ToolResult, error) {
	if idx < 0 || idx >= len(nb.Cells) {
		return errResult(fmt.Errorf("cell_number %d out of range [0, %d)", idx, len(nb.Cells))), nil
	}

	source := stringParam(params, "source", "")
	cellType := stringParam(params, "cell_type", nb.Cells[idx].CellType)

	nb.Cells[idx].Source = splitSource(source)
	nb.Cells[idx].CellType = cellType

	if err := writeNotebookFile(nb, path); err != nil {
		return errResult(err), nil
	}

	return agent.ToolResult{Content: fmt.Sprintf("replaced cell %d in %s", idx, path)}, nil
}

func notebookInsert(nb *notebook, path string, idx int, params map[string]any) (agent.ToolResult, error) {
	if idx < -1 || idx >= len(nb.Cells) {
		return errResult(fmt.Errorf("cell_number %d out of range [-1, %d)", idx, len(nb.Cells))), nil
	}

	source := stringParam(params, "source", "")
	cellType := stringParam(params, "cell_type", "code")

	newCell := cell{
		CellType: cellType,
		Source:   splitSource(source),
		Metadata: map[string]any{},
	}
	if cellType == "code" {
		newCell.Outputs = []any{}
	}

	// Insert after idx; idx == -1 means insert at the beginning.
	insertAt := idx + 1
	nb.Cells = append(nb.Cells, cell{})
	copy(nb.Cells[insertAt+1:], nb.Cells[insertAt:])
	nb.Cells[insertAt] = newCell

	if err := writeNotebookFile(nb, path); err != nil {
		return errResult(err), nil
	}

	return agent.ToolResult{Content: fmt.Sprintf("inserted cell at %d in %s", insertAt, path)}, nil
}

func notebookDelete(nb *notebook, path string, idx int) (agent.ToolResult, error) {
	if idx < 0 || idx >= len(nb.Cells) {
		return errResult(fmt.Errorf("cell_number %d out of range [0, %d)", idx, len(nb.Cells))), nil
	}

	nb.Cells = append(nb.Cells[:idx], nb.Cells[idx+1:]...)

	if err := writeNotebookFile(nb, path); err != nil {
		return errResult(err), nil
	}

	return agent.ToolResult{Content: fmt.Sprintf("deleted cell %d from %s", idx, path)}, nil
}

// splitSource converts a single string into the .ipynb line-array format.
// Each line except the last gets a trailing newline.
func splitSource(s string) []string {
	if s == "" {
		return []string{}
	}
	lines := strings.Split(s, "\n")
	result := make([]string, len(lines))
	for i, line := range lines {
		if i < len(lines)-1 {
			result[i] = line + "\n"
		} else {
			result[i] = line
		}
	}
	return result
}

func writeNotebookFile(nb *notebook, path string) error {
	data, err := json.MarshalIndent(nb, "", " ")
	if err != nil {
		return fmt.Errorf("marshalling notebook: %w", err)
	}
	// Append trailing newline for POSIX compliance.
	data = append(data, '\n')
	if err := os.WriteFile(path, data, 0o644); err != nil {
		return fmt.Errorf("writing notebook %s: %w", path, err)
	}
	return nil
}
