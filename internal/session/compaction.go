// ABOUTME: Context compaction: summarize old messages, keep recent ones
// ABOUTME: Reduces context size when approaching model token limits

package session

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/mauromedda/pi-coding-agent-go/pkg/ai"
)

// CompactionEntry records metadata about a compacted message span.
type CompactionEntry struct {
	FilesRead    []string // file paths that were read during the span
	FilesWritten []string // file paths that were written/edited during the span
	MessageCount int      // number of messages in the compacted span
}

// readTools are tool names that read files.
var readTools = map[string]bool{
	"Read": true, "Glob": true, "Grep": true,
}

// writeTools are tool names that write/edit files.
var writeTools = map[string]bool{
	"Write": true, "Edit": true, "NotebookEdit": true,
}

// ExtractFileOps scans messages for tool_use content blocks and extracts
// file paths categorized as read or written.
func ExtractFileOps(messages []ai.Message) CompactionEntry {
	entry := CompactionEntry{
		MessageCount: len(messages),
	}

	readSeen := make(map[string]bool)
	writeSeen := make(map[string]bool)

	for _, msg := range messages {
		for _, c := range msg.Content {
			if c.Type != ai.ContentToolUse || len(c.Input) == 0 {
				continue
			}

			filePath := extractFilePath(c.Input)
			if filePath == "" {
				continue
			}

			if readTools[c.Name] {
				if !readSeen[filePath] {
					readSeen[filePath] = true
					entry.FilesRead = append(entry.FilesRead, filePath)
				}
			}
			if writeTools[c.Name] {
				if !writeSeen[filePath] {
					writeSeen[filePath] = true
					entry.FilesWritten = append(entry.FilesWritten, filePath)
				}
			}
		}
	}

	return entry
}

// extractFilePath pulls the file_path field from tool input JSON.
func extractFilePath(input json.RawMessage) string {
	var args struct {
		FilePath     string `json:"file_path"`
		NotebookPath string `json:"notebook_path"`
	}
	if err := json.Unmarshal(input, &args); err != nil {
		return ""
	}
	if args.FilePath != "" {
		return args.FilePath
	}
	return args.NotebookPath
}

const keepRecentMessages = 10

// Compact summarizes older messages into a single summary message,
// keeping the most recent messages intact.
func Compact(messages []ai.Message) ([]ai.Message, string, error) {
	if len(messages) <= keepRecentMessages {
		return messages, "", nil
	}

	// Split into old and recent
	oldMessages := messages[:len(messages)-keepRecentMessages]
	recentMessages := messages[len(messages)-keepRecentMessages:]

	// Build summary from old messages
	summary := buildSummary(oldMessages)

	// Create compacted message list
	compacted := make([]ai.Message, 0, keepRecentMessages+1)
	compacted = append(compacted, ai.NewTextMessage(ai.RoleUser,
		fmt.Sprintf("[Context Summary]\n%s\n[End Summary]", summary)))
	compacted = append(compacted, ai.NewTextMessage(ai.RoleAssistant,
		"I understand the context. Let me continue from where we left off."))
	compacted = append(compacted, recentMessages...)

	return compacted, summary, nil
}

// buildSummary creates a text summary from messages.
func buildSummary(messages []ai.Message) string {
	var b strings.Builder
	for _, msg := range messages {
		b.WriteString(string(msg.Role))
		b.WriteString(": ")
		for _, c := range msg.Content {
			if c.Type == ai.ContentText {
				text := c.Text
				if len(text) > 200 {
					text = text[:200] + "..."
				}
				b.WriteString(text)
			}
		}
		b.WriteByte('\n')
	}
	return b.String()
}
