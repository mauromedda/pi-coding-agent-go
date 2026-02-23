// ABOUTME: File operation reversal logic for /revert and /undo commands
// ABOUTME: Extracts file operations from message history and reverts them via git checkout or os.Remove

package revert

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"strings"

	"github.com/mauromedda/pi-coding-agent-go/pkg/ai"
)

// fileTools are tool names that modify files on disk.
// bash is excluded: its input uses "command", not "path", making path extraction unreliable.
var fileTools = map[string]bool{
	"write": true,
	"edit":  true,
}

// FileOp represents a file operation extracted from a tool_use content block.
type FileOp struct {
	Tool string // "write", "edit", or "bash"
	Path string // file path affected
}

// toolInput is used to extract the "path" field from tool input JSON.
type toolInput struct {
	Path string `json:"path"`
}

// FindFileOps walks messages backward, extracting file-modifying tool operations.
// Returns at most maxSteps operations, most-recent first.
func FindFileOps(messages []ai.Message, maxSteps int) []FileOp {
	var ops []FileOp

	for i := len(messages) - 1; i >= 0 && len(ops) < maxSteps; i-- {
		msg := messages[i]
		if msg.Role != ai.RoleAssistant {
			continue
		}

		// Walk content blocks backward within message
		for j := len(msg.Content) - 1; j >= 0 && len(ops) < maxSteps; j-- {
			ct := msg.Content[j]
			if ct.Type != ai.ContentToolUse {
				continue
			}
			if !fileTools[ct.Name] {
				continue
			}

			var input toolInput
			if err := json.Unmarshal(ct.Input, &input); err != nil || input.Path == "" {
				continue
			}

			ops = append(ops, FileOp{
				Tool: ct.Name,
				Path: input.Path,
			})
		}
	}

	return ops
}

// RevertOps reverts file operations:
// - write (file creation): removes the file
// - edit: runs git checkout -- <path> to restore
// Returns a summary of actions taken.
func RevertOps(ops []FileOp) ([]string, error) {
	var summary []string

	for _, op := range ops {
		switch op.Tool {
		case "write":
			if err := os.Remove(op.Path); err != nil {
				if os.IsNotExist(err) {
					summary = append(summary, fmt.Sprintf("skip %s (already gone)", op.Path))
					continue
				}
				return summary, fmt.Errorf("remove %s: %w", op.Path, err)
			}
			summary = append(summary, fmt.Sprintf("removed %s", op.Path))

		case "edit":
			cmd := exec.Command("git", "checkout", "--", op.Path)
			if err := cmd.Run(); err != nil {
				summary = append(summary, fmt.Sprintf("git checkout failed for %s: %v", op.Path, err))
				continue
			}
			summary = append(summary, fmt.Sprintf("restored %s", op.Path))
		}
	}

	return summary, nil
}

// FormatSummary joins revert summaries into a single display string.
func FormatSummary(lines []string) string {
	if len(lines) == 0 {
		return "Nothing to revert."
	}
	return "Reverted:\n  " + strings.Join(lines, "\n  ")
}
