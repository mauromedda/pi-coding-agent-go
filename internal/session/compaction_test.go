// ABOUTME: Tests for compaction file tracking: extracting read/written files from tool_use blocks
// ABOUTME: Verifies CompactionEntry correctly categorizes file operations from messages

package session

import (
	"encoding/json"
	"testing"

	"github.com/mauromedda/pi-coding-agent-go/pkg/ai"
)

func TestExtractFileOps_ReadTool(t *testing.T) {
	t.Parallel()

	msgs := []ai.Message{
		{
			Role: ai.RoleAssistant,
			Content: []ai.Content{
				{
					Type:  ai.ContentToolUse,
					Name:  "Read",
					Input: json.RawMessage(`{"file_path":"/src/main.go"}`),
				},
			},
		},
	}

	entry := ExtractFileOps(msgs)

	if len(entry.FilesRead) != 1 || entry.FilesRead[0] != "/src/main.go" {
		t.Errorf("expected FilesRead=[/src/main.go], got %v", entry.FilesRead)
	}
	if len(entry.FilesWritten) != 0 {
		t.Errorf("expected no FilesWritten, got %v", entry.FilesWritten)
	}
	if entry.MessageCount != 1 {
		t.Errorf("expected MessageCount=1, got %d", entry.MessageCount)
	}
}

func TestExtractFileOps_WriteTool(t *testing.T) {
	t.Parallel()

	msgs := []ai.Message{
		{
			Role: ai.RoleAssistant,
			Content: []ai.Content{
				{
					Type:  ai.ContentToolUse,
					Name:  "Write",
					Input: json.RawMessage(`{"file_path":"/src/new.go","content":"package main"}`),
				},
			},
		},
	}

	entry := ExtractFileOps(msgs)

	if len(entry.FilesWritten) != 1 || entry.FilesWritten[0] != "/src/new.go" {
		t.Errorf("expected FilesWritten=[/src/new.go], got %v", entry.FilesWritten)
	}
}

func TestExtractFileOps_EditTool(t *testing.T) {
	t.Parallel()

	msgs := []ai.Message{
		{
			Role: ai.RoleAssistant,
			Content: []ai.Content{
				{
					Type:  ai.ContentToolUse,
					Name:  "Edit",
					Input: json.RawMessage(`{"file_path":"/src/main.go","old_string":"foo","new_string":"bar"}`),
				},
			},
		},
	}

	entry := ExtractFileOps(msgs)

	if len(entry.FilesWritten) != 1 || entry.FilesWritten[0] != "/src/main.go" {
		t.Errorf("expected FilesWritten=[/src/main.go], got %v", entry.FilesWritten)
	}
}

func TestExtractFileOps_BashTool(t *testing.T) {
	t.Parallel()

	msgs := []ai.Message{
		{
			Role: ai.RoleAssistant,
			Content: []ai.Content{
				{
					Type:  ai.ContentToolUse,
					Name:  "Bash",
					Input: json.RawMessage(`{"command":"ls -la"}`),
				},
			},
		},
	}

	entry := ExtractFileOps(msgs)

	// Bash doesn't have file_path; should not track
	if len(entry.FilesRead) != 0 {
		t.Errorf("expected no FilesRead for Bash, got %v", entry.FilesRead)
	}
	if len(entry.FilesWritten) != 0 {
		t.Errorf("expected no FilesWritten for Bash, got %v", entry.FilesWritten)
	}
}

func TestExtractFileOps_MixedMessages(t *testing.T) {
	t.Parallel()

	msgs := []ai.Message{
		ai.NewTextMessage(ai.RoleUser, "read main.go"),
		{
			Role: ai.RoleAssistant,
			Content: []ai.Content{
				{Type: ai.ContentText, Text: "Let me read that file."},
				{
					Type:  ai.ContentToolUse,
					Name:  "Read",
					Input: json.RawMessage(`{"file_path":"/src/main.go"}`),
				},
			},
		},
		ai.NewTextMessage(ai.RoleUser, "now edit it"),
		{
			Role: ai.RoleAssistant,
			Content: []ai.Content{
				{
					Type:  ai.ContentToolUse,
					Name:  "Edit",
					Input: json.RawMessage(`{"file_path":"/src/main.go","old_string":"a","new_string":"b"}`),
				},
				{
					Type:  ai.ContentToolUse,
					Name:  "Read",
					Input: json.RawMessage(`{"file_path":"/src/util.go"}`),
				},
			},
		},
	}

	entry := ExtractFileOps(msgs)

	if entry.MessageCount != 4 {
		t.Errorf("expected MessageCount=4, got %d", entry.MessageCount)
	}

	// main.go read + util.go read (deduplicated)
	if len(entry.FilesRead) != 2 {
		t.Errorf("expected 2 FilesRead, got %v", entry.FilesRead)
	}

	// main.go edited
	if len(entry.FilesWritten) != 1 {
		t.Errorf("expected 1 FilesWritten, got %v", entry.FilesWritten)
	}
}

func TestExtractFileOps_DeduplicatesFiles(t *testing.T) {
	t.Parallel()

	msgs := []ai.Message{
		{
			Role: ai.RoleAssistant,
			Content: []ai.Content{
				{
					Type:  ai.ContentToolUse,
					Name:  "Read",
					Input: json.RawMessage(`{"file_path":"/src/main.go"}`),
				},
				{
					Type:  ai.ContentToolUse,
					Name:  "Read",
					Input: json.RawMessage(`{"file_path":"/src/main.go"}`),
				},
			},
		},
	}

	entry := ExtractFileOps(msgs)

	if len(entry.FilesRead) != 1 {
		t.Errorf("expected deduplicated FilesRead to have 1 entry, got %v", entry.FilesRead)
	}
}

func TestExtractFileOps_EmptyMessages(t *testing.T) {
	t.Parallel()

	entry := ExtractFileOps(nil)

	if entry.MessageCount != 0 {
		t.Errorf("expected MessageCount=0, got %d", entry.MessageCount)
	}
	if len(entry.FilesRead) != 0 {
		t.Errorf("expected empty FilesRead, got %v", entry.FilesRead)
	}
	if len(entry.FilesWritten) != 0 {
		t.Errorf("expected empty FilesWritten, got %v", entry.FilesWritten)
	}
}
